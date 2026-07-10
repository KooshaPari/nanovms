# nanovms Testing Depth

Multi-layer test strategy: unit + property + integration + e2e.

## Layers

| Layer | Tooling | Scope | Target coverage |
|-------|---------|-------|-----------------|
| Unit | `go test` | function | 80%+ |
| Property | `gopter` | invariant | 5+ props per package |
| Integration | `testcontainers-go` | adapter | 60%+ |
| E2E | `testscript` | CLI | 1+ per command |

## Property tests (gopter)

```go
import "github.com/leanovate/gopter"

props := gopter.NewProperties(nil)

props.Property("SandboxID roundtrip", prop.ForAll(
    func(s string) bool {
        id := generateID(s)
        return validateID(id) == nil
    },
    gen.AlphaString(),
))

props.TestingRun(t)
```

## Testcontainers (integration)

```go
import "github.com/testcontainers/testcontainers-go/modules/postgres"

func TestSecretStoreWithPostgres(t *testing.T) {
    ctx := context.Background()
    pg, _ := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16"))
    defer pg.Terminate(ctx)

    url, _ := pg.ConnectionString(ctx)
    store, _ := NewPgSecretStore(url)

    // ... test cases
}
```

## E2E tests (testscript)

```bash
# tests/cmd/sandbox_create.txt
env PHENOVMS_HOME=$WORK/home
exec phenovms sandbox create test-image
stdout 'created test'
! stderr '.'

env PHENOVMS_HOME=$WORK/home
exec phenovms sandbox list
stdout 'test-image'
```

```go
// sandbox_create_test.go
func TestSandboxCreateScript(t *testing.T) {
    testscript.RunMain(m, map[string]func() int{
        "sandbox_create.txt": 1,
    })
}
```

## Coverage target

- 80% line coverage for all packages
- 100% coverage for `PortError` variants
- 100% coverage for security-critical paths
- Coverage drops in PRs fail CI

## Test helpers

```go
// internal/testutil/daemon.go
func StartTestDaemon(t *testing.T) (*Daemon, func()) {
    dir := t.TempDir()
    cfg := &Config{DataDir: dir, ListenAddr: "127.0.0.1:0"}
    d, err := NewDaemon(cfg)
    if err != nil { t.Fatal(err) }
    go d.Serve()
    return d, func() { d.Shutdown(context.Background()) }
}
```

## Mocking

```go
// Use interfaces for testability
type SandboxBackend interface {
    Create(ctx context.Context, cfg Config) (*Sandbox, error)
}

// Test impl
type fakeBackend struct {
    createFn func(context.Context, Config) (*Sandbox, error)
}

func (f *fakeBackend) Create(ctx context.Context, cfg Config) (*Sandbox, error) {
    return f.createFn(ctx, cfg)
}
```
