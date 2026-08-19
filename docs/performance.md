# nanovms Performance

Go test built-in benchmarks for the most-trafficked value types and operations.

## Running benchmarks

```bash
go test -bench=. -benchmem ./internal/domain/
go test -bench=BenchmarkSandboxConfig_New -benchtime=10s ./internal/domain/
```

## Current benchmarks (L6)

| Bench | Purpose | Target |
|-------|---------|--------|
| `BenchmarkSandboxConfig_New` | measure `SandboxConfig{}` literal allocation | < 50 ns/op, 0 alloc |
| `BenchmarkSandboxConfig_Clone` | measure value-copy of config | < 100 ns/op |
| `BenchmarkPortMapping_Slice` | measure 8-element PortMapping slice | < 500 ns/op |
| `BenchmarkSandboxID_Len` | measure ID-format fast path | < 1 ns/op |

## Memory (L19)

Per-sandbox RSS budgets are tracked at runtime. Use `go test -benchmem` to
attribute allocations back to the field level.

## CI

Benchmarks should be added to CI on PR. (Future: integrate `benchstat` for
regression detection.)
MDEEOF

# Verify
cd /Users/kooshapari/CodeProjects/Phenotype/repos/nanovms
go build ./cmd/agentctl 2>&1 | tail -3
echo "---"
echo '{"method":"sandbox.create","params":{"name":"hello"}}' | go run ./cmd/agentctl 2>&1 | head -3
echo "---"
go test -bench=. ./internal/domain/ 2>&1 | tail -10