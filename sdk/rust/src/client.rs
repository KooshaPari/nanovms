use crate::error::{NvmsError, Result};
use reqwest::Client;
use serde::de::DeserializeOwned;
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
        let client = Client::builder()
            .timeout(Duration::from_secs(30))
            .build()
            .map_err(|e| NvmsError::ClientInit(e.to_string()))?;

        Ok(Self {
            inner: client,
            base_url: base_url.into(),
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
