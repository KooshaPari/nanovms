# nanovms Extensibility

Plugins extend the daemon at runtime via two mechanisms:

1. **In-process plugins** - linked at build time (Go interfaces)
2. **Runtime plugins** - loaded via the standard `plugin` package (.so files)

## Architecture

```
   ┌──────────────────┐
   │ Daemon (registry)│  <-- plugin.Registry
   └──────────────────┘
            │
            ▼ registers
   ┌──────────────────┐    ┌──────────────────┐
   │ sample.Plugin   │    │ runtime .so file │
   │ (in-process)     │    │ (loaded at start)│
   └──────────────────┘    └──────────────────┘
```

## Plugin contract (Go interface)

```go
type Plugin interface {
    Info() Info                                       // static metadata
    Init(ctx context.Context) error                  // called once
    Shutdown(ctx context.Context) error              // called on unload
    Health(ctx context.Context) error                // periodic check
}
```

## Registry

`internal/plugin/plugin.go` provides:
- `NewRegistry() *Registry`
- `Register(ctx, p) error` - calls Init
- `UnregisterAll(ctx)` - LIFO Shutdown
- `Find(id ID) (Plugin, bool)`
- `List() []Info`
- `Len() int`

## Defining a plugin

```go
type MyPlugin struct { /* state */ }
func (p *MyPlugin) Info() plugin.Info {
    return plugin.Info{ID: "phenotype.plugin.my", Name: "My", Version: "0.1.0"}
}
func (p *MyPlugin) Init(_ context.Context) error { return nil }
func (p *MyPlugin) Shutdown(_ context.Context) error { return nil }
func (p *MyPlugin) Health(_ context.Context) error { return nil }
```

## Registering

```go
r := plugin.NewRegistry()
r.Register(ctx, &MyPlugin{})
p, _ := r.Find("phenotype.plugin.my")
```

## Runtime loading (Go .so)

```go
plug, err := plugin.Open("/path/to/my-plugin.so")
sym, err := plug.Lookup("NewPlugin")
newFn := sym.(func() plugin.Plugin)
r.Register(ctx, newFn())
```

## Thread-safety

Registry is mutex-protected; all operations are safe for concurrent use. Implementations should also be safe for concurrent invocation of `Info`, `Init`, `Shutdown`, `Health`.

## Tests

```bash
go test ./internal/plugin/...
# 3 tests pass
```
