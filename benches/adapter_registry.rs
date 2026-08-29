//! Criterion benchmark: adapter registry lookup speed.
//!
//! Benchmarks the core `PortAdapter` operations — deploy, status, and list —
//! through an in-memory adapter.  These proxy the adapter registry
//! lookup path that backends use at deploy time.

use async_trait::async_trait;
use criterion::{black_box, criterion_group, criterion_main, Criterion};
use phenotype_port_adapter_shim::adapter::{AdapterError, PortAdapter};
use phenotype_port_adapter_shim::types::{
    DeploymentId, DeploymentState, LogLine, LogOptions, PortManifest, PortStatus,
};
use std::collections::HashMap;
use tokio::sync::{mpsc, RwLock};

/// Minimal in-memory adapter for benchmarking.
#[derive(Debug)]
struct BenchAdapter {
    deployments: RwLock<HashMap<DeploymentId, DeploymentState>>,
}

impl BenchAdapter {
    fn new() -> Self {
        Self {
            deployments: RwLock::new(HashMap::new()),
        }
    }
}

#[async_trait]
impl PortAdapter for BenchAdapter {
    fn name(&self) -> &'static str {
        "bench"
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

fn bench_deploy_lookup(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();

    c.bench_function("adapter_deploy_lookup", |b| {
        let adapter = BenchAdapter::new();
        let manifest = PortManifest {
            name: "bench-app".into(),
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

        b.iter(|| {
            rt.block_on(async {
                let status = black_box(adapter.deploy(manifest.clone()).await.unwrap());
                black_box(adapter.status(&status.id).await.unwrap());
            });
        });
    });
}

fn bench_list_registrations(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();

    c.bench_function("adapter_list_registrations", |b| {
        b.iter_custom(|iters| {
            let adapter = BenchAdapter::new();
            let manifest = PortManifest {
                name: "bench-list".into(),
                image: "alpine:3.18".into(),
                cpu_shares: 512,
                memory_mib: 256,
                replicas: 1,
                env: vec![],
                command: vec![],
                ports: vec![],
                health_check_path: None,
                region: None,
            };

            // Seed the adapter with `iters` registrations.
            rt.block_on(async {
                for _ in 0..iters {
                    adapter.deploy(manifest.clone()).await.unwrap();
                }
            });

            // Measure list throughput.
            let start = std::time::Instant::now();
            rt.block_on(async {
                for _ in 0..iters {
                    black_box(adapter.list().await.unwrap());
                }
            });
            start.elapsed()
        });
    });
}

criterion_group!(benches, bench_deploy_lookup, bench_list_registrations);
criterion_main!(benches);
