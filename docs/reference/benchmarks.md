# Benchmarks

NanoVMS performance benchmarks and methodology.

## Benchmark Categories

### 1. VM Lifecycle Benchmarks

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `vm_create_cold` | Cold VM creation time | < 125ms (Firecracker) |
| `vm_create_warm` | Warm VM creation (pre-warmed) | < 10ms |
| `vm_start` | VM start time | < 50ms |
| `vm_stop` | VM stop time | < 20ms |
| `vm_delete` | VM deletion time | < 10ms |
| `vm_snapshot` | Snapshot creation | < 500ms |
| `vm_restore` | Snapshot restore | < 100ms |

### 2. Sandbox Lifecycle Benchmarks

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `sandbox_create_bwrap` | bwrap sandbox creation | < 5ms |
| `sandbox_create_firejail` | firejail sandbox creation | < 15ms |
| `sandbox_create_gvisor` | gVisor sandbox creation | < 50ms |
| `sandbox_exec` | Command execution in sandbox | < 10ms |

### 3. WASM Benchmarks

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `wasm_cold_start` | Cold WASM module start | < 1ms |
| `wasm_warm_start` | Warm WASM module start | < 100μs |
| `wasm_exec` | WASM execution | < 10μs per instruction |

### 4. Resource Utilization

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `memory_per_vm` | Memory overhead per MicroVM | < 5MB |
| `memory_per_sandbox` | Memory overhead per sandbox | < 1MB |
| `cpu_overhead` | CPU overhead | < 1% |
| `disk_overhead` | Disk overhead per VM | < 10MB |

### 5. Network Benchmarks

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `net_throughput` | Network throughput | > 5 Gbps |
| `net_latency` | Network latency (VM to VM) | < 100μs |
| `net_pps` | Packets per second | > 1M pps |

### 6. I/O Benchmarks

| Benchmark | Description | Target |
|-----------|-------------|--------|
| `io_sequential_read` | Sequential read throughput | > 3 GB/s |
| `io_sequential_write` | Sequential write throughput | > 1 GB/s |
| `io_random_read` | Random read IOPS | > 100K IOPS |
| `io_random_write` | Random write IOPS | > 50K IOPS |

## Running Benchmarks

```bash
# Run all benchmarks
nanovms benchmark run

# Run specific category
nanovms benchmark run --category vm
nanovms benchmark run --category sandbox
nanovms benchmark run --category wasm

# Run with comparison to baseline
nanovms benchmark run --compare --baseline=/path/to/baseline.json

# Run with JSON output
nanovms benchmark run --json --output=/tmp/benchmarks.json
```

## Benchmark Environment

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| RAM | 16GB | 32GB+ |
| Storage | SSD | NVMe |
| Network | 1Gbps | 10Gbps |

### Software Requirements

| Component | Version |
|-----------|---------|
| Linux Kernel | 5.15+ |
| KVM | Latest |
| Firecracker | 1.5+ |
| Go | 1.21+ |

## Benchmark Results Format

```json
{
  "version": "1.0",
  "timestamp": "2026-01-15T10:30:00Z",
  "environment": {
    "os": "Linux 6.1.0",
    "arch": "amd64",
    "cpu": "AMD Ryzen 9 5950X",
    "memory": "65536MB",
    "kernel": "Linux 6.1.0"
  },
  "results": [
    {
      "name": "vm_create_cold",
      "iterations": 100,
      "mean_ms": 118.5,
      "median_ms": 115.2,
      "p50_ms": 115.2,
      "p90_ms": 125.0,
      "p99_ms": 140.0,
      "min_ms": 95.0,
      "max_ms": 200.0,
      "stddev_ms": 15.2
    }
  ]
}
```

## CI/CD Integration

Benchmarks run automatically in CI/CD on every PR and can block merges if performance regresses.

```yaml
# .github/workflows/benchmark.yml
name: Benchmark

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  benchmark:
    runs-on: benchmark-runner
    steps:
      - uses: actions/checkout@v4
      
      - name: Run benchmarks
        run: nanovms benchmark run --json --output=results.json
      
      - name: Compare with baseline
        run: |
          nanovms benchmark compare \
            --current=results.json \
            --baseline=${{ github.event.before }}/results.json \
            --threshold=5%
      
      - name: Store results
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'customJson'
          outputFilePath: results.json
          auto-push: true
          alertThreshold: '10%'
```

## Continuous Benchmarking

Benchmark results are stored and visualized over time to track performance trends.

| Metric | Alert Threshold |
|---------|----------------|
| VM creation time | +10% regression |
| Memory overhead | +20% regression |
| CPU overhead | +5% regression |
