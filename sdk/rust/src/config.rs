use crate::error::{NvmsError, Result};
use figment::{
    providers::{Env, Format, Serialized, Toml},
    Figment,
};
use serde::{Deserialize, Serialize};
use std::path::Path;

fn default_timeout_seconds() -> u64 {
    30
}

/// SDK configuration loaded through figment providers.
#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
pub struct NvmsConfig {
    pub base_url: String,
    #[serde(default = "default_timeout_seconds")]
    pub timeout_seconds: u64,
}

impl NvmsConfig {
    /// Create config directly from a base URL.
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            timeout_seconds: default_timeout_seconds(),
        }
    }

    /// Load configuration from a TOML file, optionally overridden by env vars.
    pub fn from_toml_file(path: impl AsRef<Path>) -> Result<Self> {
        Figment::from(Serialized::defaults(Self::new("http://127.0.0.1:8080")))
            .merge(Toml::file(path))
            .merge(Env::prefixed("NVMS_"))
            .extract()
            .map_err(|e| NvmsError::Config(e.to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::NvmsConfig;
    use std::fs;
    use tempfile::NamedTempFile;

    #[test]
    fn loads_toml_config() {
        let file = NamedTempFile::new().expect("temp file");
        fs::write(
            file.path(),
            r#"
base_url = "https://nvms.test"
timeout_seconds = 42
"#,
        )
        .expect("write config");

        let config = NvmsConfig::from_toml_file(file.path()).expect("load config");
        assert_eq!(config.base_url, "https://nvms.test");
        assert_eq!(config.timeout_seconds, 42);
    }
}
