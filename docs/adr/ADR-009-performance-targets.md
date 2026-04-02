# ADR-009: Performance Targets

## Status

Accepted

## Context

NanoVMS targets workloads requiring:
- Fast VM startup
- Low resource overhead
- High density
- Predictable performance

## Decision

### Performance Targets by Tier

```
┌────────────────────────────────────────────────────────────────────┐
│                    Performance Target Matrix                           │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Tier  │ Startup (cold) │ Startup (warm) │ Memory   │ CPU    │ Density│
│  ──────┼────────────────┼────────────────┼──────────┼────────┼────────│
│  VFIO  │ 30-60s        │ <5s            │ <1MB     │ <1%    │ 1-2   │
│  MicroVM│ <100ms        │ <10ms          │ <5MB     │ <1%    │ 150/s │
│  Kata   │ 1-2s          │ 200ms          │ 50MB     │ 3%     │ 50    │
│  gVisor │ 100ms         │ 50ms           │ 50-150MB │ 5%     │ 100   │
│  bwrap  │ 5-10ms        │ <2ms           │ <1MB     │ <1%    │ 500   │
│  unshare│ 2-5ms         │ <1ms           │ <1MB     │ <0.5%  │ 1000  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Benchmark Definitions

#### VM Creation

```bash
# Benchmark: Create 100 VMs sequentially
# Target: <100ms per VM (MicroVM tier)
hyperfine \
  --prepare 'nanovms vm delete --all' \
  --min-runs 10 \
  -- 'nanovms vm create --name=test-{} --flavor=microvm'

# Expected output:
# Time (mean): 87.4ms ± 2.3ms
```

#### VM Deletion

```bash
# Benchmark: Delete 100 VMs
# Target: <50ms per VM
hyperfine \
  --min-runs 10 \
  -- 'nanovms vm delete test-{}'
```

#### Exec Latency

```bash
# Benchmark: Execute command in VM
# Target: <10ms round-trip (same-host)
hyperfine \
  --min-runs 100 \
  -- 'nanovms exec test-vm -- echo "hello"'

# Expected output:
# Time (mean): 8.2ms ± 0.5ms
```

#### Memory Overhead

```bash
# Benchmark: Memory per VM instance
# Target: <5MB per MicroVM
ps aux | grep nanovms | awk '{sum += $6} END {print sum/1024 " MB"}'
```

#### Startup Time Distribution

```bash
# Benchmark: Startup time histogram
# Target: P99 < 150ms
for i in {1..1000}; do
  start=$(date +%s%N)
  nanovms vm start test-vm
  end=$(date +%s%N)
  echo "$((($end - $start) / 1000000)) ms"
done | sort -n | awk 'NR==990 {print "P99: " $1 "ms"}'
```

### Performance Test Suite

```go
// benchmarks/performance_test.go
package benchmarks

func BenchmarkVMCreate(b *testing.B) {
    for i := 0; i < b.N; i++ {
        vm, err := adapter.CreateVM(context.Background(), &domain.VMConfig{
            Name:   fmt.Sprintf("bench-%d", i),
            Flavor: domain.FlavorMicroVM,
        })
        if err != nil {
            b.Fatal(err)
        }
        defer adapter.DeleteVM(context.Background(), vm.ID)
    }
}

func BenchmarkVMStart(b *testing.B) {
    vm, _ := adapter.CreateVM(ctx, &config)
    defer adapter.DeleteVM(ctx, vm.ID)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if err := adapter.StartVM(ctx, vm.ID); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkVMExec(b *testing.B) {
    vm, _ := adapter.CreateVM(ctx, &config)
    adapter.StartVM(ctx, vm.ID)
    defer adapter.DeleteVM(ctx, vm.ID)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var stdout, stderr bytes.Buffer
        err := adapter.ExecVM(ctx, vm.ID, []string{"echo", "hello"}, nil, &stdout, &stderr)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Profiling Points

```bash
# CPU profiling
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Memory profiling
curl http://localhost:6060/debug/pprof/heap > mem.prof

# Goroutine profiling
curl http://localhost:6060/debug/pprof/goroutine > goroutines.prof

# Trace
curl http://localhost:6060/debug/pprof/trace?seconds=10 > trace.out
```

### Performance Baselines

| Component | Baseline | Target | Measurement |
|-----------|----------|--------|------------|
| CLI startup | 50ms | <20ms | `time nanovms --version` |
| VM create (MicroVM) | 125ms | <100ms | benchmark |
| VM create (gVisor) | 150ms | <100ms | benchmark |
| VM start (resume) | 50ms | <10ms | benchmark |
| Exec round-trip | 15ms | <10ms | benchmark |
| Memory per MicroVM | 5MB | <5MB | ps aux |
| Memory per gVisor | 150MB | <50MB | ps aux |
| Disk per VM | 500MB | <10MB | du -sh |
| Concurrent VMs | 50 | 150 | benchmark |

## Consequences

### Positive
- Clear performance requirements
- Measurable success criteria
- Benchmark-driven development

### Negative
- Platform-specific optimizations may be needed
- Hardware-dependent baselines
- Continuous performance regression prevention
