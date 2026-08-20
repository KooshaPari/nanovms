use criterion::{criterion_group, criterion_main, Criterion};

fn ffi_throughput_benchmark(c: &mut Criterion) {
    c.bench_function("ffi_throughput", |b| b.iter(|| {
        // TODO: benchmark code here
    }));
}

criterion_group!(benches, ffi_throughput_benchmark);
criterion_main!(benches);
