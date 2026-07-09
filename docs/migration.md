# nanovms Migration Guide

## v0.1.x -> v0.2.x

### Breaking changes

- `SandboxPort.Create` now takes a `context.Context` (was sync)
- `SandboxConfig` no longer has a default `Image` field

### Migration steps

```go
// Before
sb, err := adapter.Create(cfg)

// After
sb, err := adapter.Create(ctx, cfg)
```

### Codemod

```bash
# Use go fix to update call sites
go run golang.org/x/tools/cmd/goimports@latest -w .
go fix ./...
```

## Support window

- Latest 2 minor versions
- Security patches: latest only
