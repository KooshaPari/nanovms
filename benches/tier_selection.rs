//! Criterion benchmark: tier selection policy evaluation speed.
//!
//! Measures the throughput of `DeploymentState` predicates used by the
//! port-adapter tier selection logic — `is_terminal()` and `is_running()`.

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use phenotype_port_adapter_shim::types::DeploymentState;

fn bench_is_terminal(c: &mut Criterion) {
    let states = [
        DeploymentState::Deploying,
        DeploymentState::Running,
        DeploymentState::Stopped,
        DeploymentState::Terminated,
        DeploymentState::Degraded,
        DeploymentState::Failed,
    ];

    c.bench_function("deployment_state_is_terminal", |b| {
        b.iter(|| {
            for s in &states {
                black_box(s.is_terminal());
            }
        });
    });
}

fn bench_is_running(c: &mut Criterion) {
    let states = [
        DeploymentState::Deploying,
        DeploymentState::Running,
        DeploymentState::Stopped,
        DeploymentState::Terminated,
        DeploymentState::Degraded,
        DeploymentState::Failed,
    ];

    c.bench_function("deployment_state_is_running", |b| {
        b.iter(|| {
            for s in &states {
                black_box(s.is_running());
            }
        });
    });
}

criterion_group!(benches, bench_is_terminal, bench_is_running);
criterion_main!(benches);
