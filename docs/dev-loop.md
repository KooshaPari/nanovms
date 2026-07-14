# nanovms Dev Loop

## Toolchain

- Go 1.21+ (managed via mise or go)
- Rust 1.75+ (for FFI extensions)
- mise: `mise.toml` pins all versions

## Container

`.devcontainer/devcontainer.json` provides:
- go:1.21-bookworm base
- rust 1.75 feature
- golangci-lint pre-installed
- Port 8080 forwarded (daemon)

## Common commands

```bash
# Build everything
go build ./...

# Run all tests
go test ./...

# Lint (config: .github/golangci.yml)
golangci-lint run ./...

# Bench
go test -bench=. ./internal/domain/
```

## Hot-reload

```bash
go install github.com/cosmtrek/air@latest
air
```
MDEEOF

# Verify
cd /Users/kooshapari/CodeProjects/Phenotype/repos/nanovms
go build ./... 2>&1 | tail -3
go test ./... 2>&1 | grep -E "FAIL|ok\s" | tail -3
echo "---"
ls docs/api/ .devcontainer/ 2>&1