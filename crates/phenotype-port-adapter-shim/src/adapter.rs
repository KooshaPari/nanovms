//! Port-adapter trait definitions.
//!
//! Two levels of abstraction:
//!
//! 1. [`PortAdapter`] — general deploy/status/stop/logs for any backend.
//! 2. [`NvmsAdapter`] — NanoVMS-specific operations (MicroVM lifecycle,
//!    disk attach, network config).
//!
//! Implementors: byteport-engine, PhenoCompose, NanoVMS daemon, OmniRoute
//! executor.

use async_trait::async_trait;
use tokio::sync::mpsc;

use crate::types::{
    DeploymentId, LogLine, LogOptions, PortManifest, PortStatus,
};

/// Errors from port-adapter operations.
#[derive(Debug, thiserror::Error)]
pub enum AdapterError {
    #[error("operation not implemented: {0}")]
    NotImplemented(&'static str),
    #[error("deployment failed: {0}")]
    DeploymentFailed(String),
    #[error("not found: {0}")]
    NotFound(DeploymentId),
    #[error("invalid manifest: {0}")]
    InvalidManifest(String),
    #[error("adapter unavailable: {0}")]
    Unavailable(String),
    #[error(transparent)]
    Other(Box<dyn std::error::Error + Send + Sync + 'static>),
}

impl From<String> for AdapterError {
    fn from(s: String) -> Self {
        AdapterError::DeploymentFailed(s)
    }
}

// ---------------------------------------------------------------------------
// PortAdapter (general)
// ---------------------------------------------------------------------------

/// General port-adapter interface implemented by every deployment backend.
///
/// Methods mirror `byteport_engine::Engine` for compatibility. The adapter
/// is the stable cross-repo contract; concrete backends implement this
/// trait (not the internal `Engine` trait directly).
#[async_trait]
pub trait PortAdapter: Send + Sync + std::fmt::Debug {
    /// Human-readable backend name (e.g. `"docker"`, `"firecracker"`).
    fn name(&self) -> &'static str;

    /// Deploy a new service from a manifest.
    async fn deploy(&self, manifest: PortManifest) -> Result<PortStatus, AdapterError>;

    /// Poll the current status of a deployment.
    async fn status(&self, id: &DeploymentId) -> Result<PortStatus, AdapterError>;

    /// Stop / destroy a deployment.
    async fn stop(&self, id: &DeploymentId, destroy: bool) -> Result<(), AdapterError>;

    /// Stream logs from a deployment.
    async fn logs(
        &self,
        id: &DeploymentId,
        opts: LogOptions,
    ) -> Result<mpsc::Receiver<Result<LogLine, AdapterError>>, AdapterError>;

    /// List all deployments managed by this adapter.
    async fn list(&self) -> Result<Vec<PortStatus>, AdapterError>;
}

// ---------------------------------------------------------------------------
// NvmsAdapter (NanoVMS-specific)
// ---------------------------------------------------------------------------

/// NanoVMS-specific adapter operations beyond general deployment.
///
/// These cover MicroVM lifecycle management, disk snapshotting, virtual
/// networking, and direct Firecracker / KVM / QEMU control.
#[async_trait]
pub trait NvmsAdapter: PortAdapter {
    /// Provision a new MicroVM (Firecracker / KVM) with the given spec.
    async fn provision_microvm(&self, spec: VmSpec) -> Result<VmId, AdapterError>;

    /// Get MicroVM state.
    async fn microvm_status(&self, id: &VmId) -> Result<VmStatus, AdapterError>;

    /// Attach a disk image to a running MicroVM.
    async fn attach_disk(&self, vm_id: &VmId, disk: DiskSpec) -> Result<(), AdapterError>;

    /// Configure virtual networking for a MicroVM.
    async fn configure_network(&self, vm_id: &VmId, net: NetworkConfig) -> Result<(), AdapterError>;
}

// ---------------------------------------------------------------------------
// NanoVMS-specific types
// ---------------------------------------------------------------------------

/// MicroVM identifier (opaque).
#[derive(Debug, Clone, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
#[serde(transparent)]
pub struct VmId(pub String);

/// Spec for provisioning a MicroVM.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct VmSpec {
    pub vcpu_count: u16,
    pub memory_mib: u64,
    pub root_disk_mib: u64,
    pub kernel_image: Option<String>,
    pub initrd: Option<String>,
    pub boot_args: Vec<String>,
}

/// MicroVM runtime status.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct VmStatus {
    pub id: VmId,
    pub state: VmState,
    pub uptime_seconds: u64,
    pub ip_address: Option<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum VmState {
    Booting,
    Running,
    Stopped,
    Failed,
}

/// Disk attachment spec.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct DiskSpec {
    /// Path or volume ID of the disk image.
    pub source: String,
    /// Read-only mount.
    pub read_only: bool,
    /// Device identifier (e.g. `/dev/vdb`).
    pub device: Option<String>,
}

/// Virtual network configuration.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct NetworkConfig {
    pub tap_interface: Option<String>,
    pub allow_mmds: bool,
    pub rate_limiter: Option<RateLimiter>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct RateLimiter {
    pub bandwidth_mbps: u64,
    pub ops_per_sec: u64,
}
