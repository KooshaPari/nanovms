//! Shared value types for the port-adapter interface.
//!
//! These types define the contract between BytePort and deployment backends
//! (Docker, Firecracker via NanoVMS, Kubernetes, AWS ECS, etc.).

use serde::{Deserialize, Serialize};

/// Opaque deployment identifier assigned by the backend.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(transparent)]
pub struct DeploymentId(pub String);

impl std::fmt::Display for DeploymentId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        self.0.fmt(f)
    }
}

/// High-level deployment state (engine-agnostic).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum DeploymentState {
    Deploying,
    Running,
    Stopped,
    Terminated,
    Degraded,
    Failed,
}

impl DeploymentState {
    pub fn is_terminal(self) -> bool {
        matches!(self, Self::Terminated | Self::Failed)
    }
    pub fn is_running(self) -> bool {
        self == Self::Running
    }
}

/// Full status returned by [`super::adapter::PortAdapter::status`].
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortStatus {
    pub id: DeploymentId,
    pub state: DeploymentState,
    pub urls: Vec<String>,
    pub ports: Vec<u16>,
    pub message: Option<String>,
    pub engine_detail: Option<serde_json::Value>,
}

/// Manifest passed to [`super::adapter::PortAdapter::deploy`].
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortManifest {
    pub name: String,
    pub image: String,
    pub cpu_shares: u64,
    pub memory_mib: u64,
    pub replicas: u32,
    pub env: Vec<EnvVar>,
    pub command: Vec<String>,
    pub ports: Vec<PortMapping>,
    pub health_check_path: Option<String>,
    pub region: Option<String>,
}

/// Environment variable.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvVar {
    pub key: String,
    pub value: String,
}

/// Port mapping.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortMapping {
    pub container_port: u16,
    pub host_port: Option<u16>,
    pub protocol: PortProtocol,
}

/// Network protocol.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum PortProtocol {
    Tcp,
    Udp,
}

impl Default for PortProtocol {
    fn default() -> Self {
        Self::Tcp
    }
}

/// A single log line.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogLine {
    pub line: String,
    pub stream: LogStream,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

/// Log stream identifier.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum LogStream {
    Stdout,
    Stderr,
}

/// Options for [`super::adapter::PortAdapter::logs`].
#[derive(Debug, Clone)]
pub struct LogOptions {
    pub tail: Option<u32>,
    pub follow: bool,
    pub since: Option<chrono::DateTime<chrono::Utc>>,
}

impl Default for LogOptions {
    fn default() -> Self {
        Self {
            tail: Some(100),
            follow: false,
            since: None,
        }
    }
}
