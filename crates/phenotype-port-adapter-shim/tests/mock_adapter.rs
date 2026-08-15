//! In-memory mock port adapter for testing.
//!
//! Mirrors `byteport_engine::adapters::mock::MockEngine` but implements the
//! `phenotype_port_adapter_shim` traits, proving the interface contract
//! is implementable.

use async_trait::async_trait;
use std::collections::HashMap;
use tokio::sync::{mpsc, RwLock};

use phenotype_port_adapter_shim::adapter::{AdapterError, PortAdapter};
use phenotype_port_adapter_shim::types::{
    DeploymentId, DeploymentState, LogLine, LogOptions, PortManifest, PortStatus,
};

/// In-memory mock backed by a `HashMap`.
#[derive(Debug)]
pub struct MockPortAdapter {
    deployments: RwLock<HashMap<DeploymentId, DeploymentState>>,
}

impl Default for MockPortAdapter {
    fn default() -> Self {
        Self::new()
    }
}

impl MockPortAdapter {
    pub fn new() -> Self {
        Self {
            deployments: RwLock::new(HashMap::new()),
        }
    }
}

#[async_trait]
impl PortAdapter for MockPortAdapter {
    fn name(&self) -> &'static str {
        "mock"
    }

    async fn deploy(&self, manifest: PortManifest) -> Result<PortStatus, AdapterError> {
        let id = DeploymentId(uuid::Uuid::new_v4().to_string());
        let mut map = self.deployments.write().await;
        map.insert(id.clone(), DeploymentState::Running);
        Ok(PortStatus {
            id,
            state: DeploymentState::Running,
            urls: vec![],
            ports: vec![],
            message: Some(format!("deployed {}", manifest.name)),
            engine_detail: None,
        })
    }

    async fn status(&self, id: &DeploymentId) -> Result<PortStatus, AdapterError> {
        let map = self.deployments.read().await;
        let state = map.get(id).copied().ok_or(AdapterError::NotFound(id.clone()))?;
        Ok(PortStatus {
            id: id.clone(),
            state,
            urls: vec![],
            ports: vec![],
            message: None,
            engine_detail: None,
        })
    }

    async fn stop(&self, id: &DeploymentId, destroy: bool) -> Result<(), AdapterError> {
        let mut map = self.deployments.write().await;
        if destroy {
            map.remove(id);
        } else {
            map.insert(id.clone(), DeploymentState::Stopped);
        }
        Ok(())
    }

    async fn logs(
        &self,
        _id: &DeploymentId,
        _opts: LogOptions,
    ) -> Result<mpsc::Receiver<Result<LogLine, AdapterError>>, AdapterError> {
        let (tx, rx) = mpsc::channel(16);
        drop(tx);
        Ok(rx)
    }

    async fn list(&self) -> Result<Vec<PortStatus>, AdapterError> {
        let map = self.deployments.read().await;
        Ok(map
            .iter()
            .map(|(id, state)| PortStatus {
                id: id.clone(),
                state: *state,
                urls: vec![],
                ports: vec![],
                message: None,
                engine_detail: None,
            })
            .collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use phenotype_port_adapter_shim::adapter::PortAdapter;

    #[tokio::test]
    async fn deploy_returns_running_status() {
        let adapter = MockPortAdapter::new();
        let manifest = PortManifest {
            name: "test".into(),
            image: "nginx:alpine".into(),
            cpu_shares: 1024,
            memory_mib: 512,
            replicas: 1,
            env: vec![],
            command: vec![],
            ports: vec![],
            health_check_path: None,
            region: None,
        };
        let status = adapter.deploy(manifest).await.expect("deploy");
        assert_eq!(status.state, DeploymentState::Running);
        assert!(status.message.unwrap().contains("deployed test"));
    }

    #[tokio::test]
    async fn stop_without_destroy_sets_stopped() {
        let adapter = MockPortAdapter::new();
        let manifest = PortManifest {
            name: "s".into(),
            image: "i".into(),
            cpu_shares: 1024,
            memory_mib: 512,
            replicas: 1,
            env: vec![],
            command: vec![],
            ports: vec![],
            health_check_path: None,
            region: None,
        };
        let status = adapter.deploy(manifest).await.unwrap();
        adapter.stop(&status.id, false).await.unwrap();
        let st = adapter.status(&status.id).await.unwrap();
        assert_eq!(st.state, DeploymentState::Stopped);
    }

    #[tokio::test]
    async fn destroy_removes_deployment() {
        let adapter = MockPortAdapter::new();
        let manifest = PortManifest {
            name: "d".into(),
            image: "i".into(),
            cpu_shares: 1024,
            memory_mib: 512,
            replicas: 1,
            env: vec![],
            command: vec![],
            ports: vec![],
            health_check_path: None,
            region: None,
        };
        let status = adapter.deploy(manifest).await.unwrap();
        adapter.stop(&status.id, true).await.unwrap();
        let err = adapter.status(&status.id).await.unwrap_err();
        assert!(matches!(err, AdapterError::NotFound(_)));
    }

    #[tokio::test]
    async fn list_returns_all() {
        let adapter = MockPortAdapter::new();
        let manifest = PortManifest {
            name: "l".into(),
            image: "i".into(),
            cpu_shares: 1024,
            memory_mib: 512,
            replicas: 1,
            env: vec![],
            command: vec![],
            ports: vec![],
            health_check_path: None,
            region: None,
        };
        adapter.deploy(manifest).await.unwrap();
        let all = adapter.list().await.unwrap();
        assert_eq!(all.len(), 1);
    }
}
