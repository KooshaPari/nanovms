# ADR-007: Testing Strategy

## Status

Accepted

## Context

NanoVMS needs comprehensive testing across:
- Unit tests
- Integration tests
- Performance benchmarks
- Game automation testing

## Decision

### Test Pyramid

```
         ┌─────────────┐
         │   E2E Tests │     ~10%
         └──────┬──────┘
         ┌──────┴──────┐
         │ Integration │    ~30%
         └──────┬──────┘
         ┌──────┴──────┐
         │ Unit Tests  │    ~60%
         └─────────────┘
```

### Unit Tests

```go
// pkg/hypervisor/firecracker_test.go
func TestFirecrackerCreate(t *testing.T) {
    vm, err := NewFirecrackerVM(Config{
        Name:   "test",
        VCPUs:  2,
        Memory: "512M",
    })
    require.NoError(t, err)
    require.NotNil(t, vm)
}
```

### Integration Tests

```go
// test/integration/smoke_test.go
func TestVM lifecycle(t *testing.T) {
    // Create -> Start -> Stop -> Delete
    vm := client.VM.Create(ctx, &CreateRequest{Name: "test"})
    require.NotEmpty(t, vm.ID)

    err := vm.Start(ctx)
    require.NoError(t, err)

    status, _ := vm.Status(ctx)
    require.Equal(t, "running", status)

    err = vm.Stop(ctx)
    require.NoError(t, err)
}
```

### Performance Benchmarks

```go
// benchmarks/vm_test.go
func BenchmarkFirecrackerStart(b *testing.B) {
    for i := 0; i < b.N; i++ {
        vm := NewFirecrackerVM(config)
        vm.Start(ctx)
        vm.Stop(ctx)
    }
}
```

### Game Automation Tests

```go
// test/game/automation_test.go
func TestGameParallelExecution(t *testing.T) {
    // Start 8 game VMs in parallel
    vms := make([]*GameVM, 8)
    for i := range vms {
        vms[i] = game.CreateVM(ctx, &GameConfig{
            Headless: true,
            Mods:    []string{"bepinex"},
        })
    }

    // Wait for all to start
    for _, vm := range vms {
        vm.WaitForReady(ctx, 30*time.Second)
    }

    // Run tests
    for _, vm := range vms {
        vm.RunTest(ctx, "regression")
    }
}
```

## Test Infrastructure

### CI/CD Pipeline

```yaml
# .github/workflows/test.yml
jobs:
  test:
    - unit-tests
    - integration-tests (multi-platform)
    - benchmark-comparison
    - game-automation-tests

  security:
    - vulnerability-scan
    - penetration-tests
```

### Test Containers

```go
// Use testcontainers for integration tests
func TestWithDatabase(t *testing.T) {
    req := testcontainers.ContainerRequest{
        Image: "postgres:16-alpine",
    }
    container, _ := testcontainers.GenericContainer(ctx, req)

    // Use container for tests
    dsn := container.MustConnectionString()

    // Cleanup
    defer container.Terminate(ctx)
}
```

## Coverage Targets

| Component | Target |
|----------|--------|
| Core | 90% |
| Adapters | 80% |
| CLI | 70% |
| Overall | 85% |

## Consequences

### Positive
- Fast feedback loop (unit tests)
- Real environment testing (integration)
- Performance regression detection
- Game automation validation

### Negative
- Complex test infrastructure
- Multi-platform testing overhead
