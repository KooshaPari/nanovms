//! Criterion benchmark: FFI mutex throughput.
//!
//! Measures lock/unlock cycle cost for the [`ffi_utils::FfiMutex`] type,
//! which is the thin `parking_lot::Mutex` alias used at cdylib FFI
//! boundaries.

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use ffi_utils::FfiMutex;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

fn ffi_mutex_lock_unlock(c: &mut Criterion) {
    let counter = AtomicU64::new(0);
    let mutex = FfiMutex::new(&counter);

    c.bench_function("ffi_mutex_lock_unlock", |b| {
        b.iter(|| {
            let _guard = black_box(mutex.lock());
            black_box(counter.fetch_add(1, Ordering::Relaxed));
        });
    });
}

fn ffi_mutex_contention(c: &mut Criterion) {
    let mutex = Arc::new(FfiMutex::new(0u64));

    c.bench_function("ffi_mutex_contention_4threads", |b| {
        let rt = tokio::runtime::Runtime::new().unwrap();
        b.iter(|| {
            let mut handles = Vec::with_capacity(4);
            for _ in 0..4 {
                let m = Arc::clone(&mutex);
                handles.push(tokio::spawn(async move {
                    let _guard = m.lock();
                    // Simulate minimal critical section work.
                    black_box(*_guard);
                }));
            }
            rt.block_on(async {
                for h in handles {
                    h.await.unwrap();
                }
            });
        });
    });
}

criterion_group!(benches, ffi_mutex_lock_unlock, ffi_mutex_contention);
criterion_main!(benches);
