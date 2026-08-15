//! # phenotype-port-adapter-shim
//!
//! v0.3 — Canonical port-adapter traits shared by BytePort, NanoVMS, and
//! OmniRoute. Defines the stable interface contract that every deployment
//! backend implements.
//!
//! ## Structure
//!
//! | Module | Description |
//! |--------|-------------|
//! | [`types`] | Value types shared across all adapters |
//! | [`adapter`] | `PortAdapter` trait (general) + `NvmsAdapter` (NanoVMS-specific) |
//!
//! ## Quick start
//!
//! ```no_run
//! use phenotype_port_adapter_shim::adapter::PortAdapter;
//! use phenotype_port_adapter_shim::types::{PortManifest, PortStatus};
//!
//! async fn example(adapter: &impl PortAdapter) -> PortStatus {
//!     let manifest = PortManifest {
//!         name: "my-app".into(),
//!         image: "nginx:alpine".into(),
//!         cpu_shares: 1024,
//!         memory_mib: 512,
//!         replicas: 1,
//!         env: vec![],
//!         command: vec![],
//!         ports: vec![],
//!         health_check_path: None,
//!         region: None,
//!     };
//!     adapter.deploy(manifest).await.unwrap()
//! }
//! ```

pub mod adapter;
pub mod types;
