use crate::config::NvmsConfig;
use crate::error::{NvmsError, Result};
use reqwest::Client;
use serde::de::DeserializeOwned;
use std::path::Path;
use std::time::Duration;

/// Async HTTP client for the NanoVMS REST API.
#[derive(Debug, Clone)]
pub struct NvmsClient {
    inner: Client,
    base_url: String,
}

impl NvmsClient {
    /// Create a new client pointing at the given NanoVMS API base URL.
    pub async fn new(base_url: impl Into<String>) -> Result<Self> {
        Self::from_config(NvmsConfig::new(base_url))
    }

    /// Create a client from a TOML configuration file.
    pub async fn from_toml_file(path: impl AsRef<Path>) -> Result<Self> {
        let config = NvmsConfig::from_toml_file(path)?;
        Self::from_config(config)
    }

    /// Create a client from an already loaded SDK config.
    pub fn from_config(config: NvmsConfig) -> Result<Self> {
        let client = Client::builder()
            .timeout(Duration::from_secs(config.timeout_seconds))
            .build()
            .map_err(|e| NvmsError::ClientInit(e.to_string()))?;

        Ok(Self {
            inner: client,
            base_url: config.base_url,
        })
    }

    /// Perform a GET request and deserialize the JSON response.
    pub async fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T> {
        let url = format!("{}/api/v1{}", self.base_url, path);
        let resp = self
            .inner
            .get(&url)
            .send()
            .await
            .map_err(|e| NvmsError::RequestFailed(e.to_string()))?;

        if !resp.status().is_success() {
            return Err(NvmsError::HttpStatus {
                status: resp.status().as_u16(),
                body: resp.text().await.unwrap_or_default(),
            });
        }

        resp.json::<T>()
            .await
            .map_err(|e| NvmsError::Deserialize(e.to_string()))
    }

    /// List all VMs.
    pub async fn list_vms(&self) -> Result<Vec<crate::models::Vm>> {
        self.get("/vms").await
    }
}

#[cfg(test)]
mod tests {
    use super::NvmsClient;
    use std::fs;
    use tempfile::NamedTempFile;

    #[tokio::test]
    async fn builds_client_from_toml_file() {
        let file = NamedTempFile::new().expect("temp file");
        fs::write(
            file.path(),
            r#"
base_url = "https://api.nvms.test"
timeout_seconds = 5
"#,
        )
        .expect("write config");

        let client = NvmsClient::from_toml_file(file.path())
            .await
            .expect("client from config");
        assert_eq!(client.base_url, "https://api.nvms.test");
    }
}
