# AGENTS.md — devenv-abstraction

## Project Overview

- **Name**: devenv-abstraction
- **Description**: Docker-alternative VM stack with OCI/sandbox support. Hexagonal architecture for container runtime abstraction across Mac, Windows, Linux
- **Location**: `/Users/kooshapari/CodeProjects/Phenotype/repos/devenv-abstraction`
- **Language Stack**: Go 1.22+
- **Published**: Private (Phenotype org)

## Quick Start

```bash
# Clone and setup
git clone https://github.com/KooshaPari/devenv-abstraction.git
cd devenv-abstraction
go mod download

# Build
go build ./...

# Run tests
go test ./...

# Run CLI
go run ./cmd/devenv-abstraction

# Build documentation
cd docs && npm install && npm run docs:build
```

## Architecture

### Hexagonal Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Ports (Interfaces)                    │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │ SandboxPort     │  │ RuntimePort     │  │ ImagePort   │ │
│  └────────┬────────┘  └────────┬────────┘  └──────┬──────┘ │
└───────────┼─────────────────────┼──────────────────┼────────┘
            │                     │                  │
┌───────────▼─────────────────────▼──────────────────▼────────┐
│                      Adapters                               │
│  ┌──────────┐  ┌───────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Mac      │  │ Windows   │  │ Linux    │  │ WASM     │ │
│  │ (Lima)   │  │ (WSL2)    │  │ (Native) │  │ (Wasmtime)│ │
│  └──────────┘  └───────────┘  └──────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Platform Support

| Platform | Backend | Status | Notes |
|----------|---------|--------|-------|
| macOS | Lima/Colima + vz | ✅ Active | Primary dev environment |
| Windows | WSL2 + gVisor | ✅ Active | Primary Windows path |
| Linux | Native + gVisor | ✅ Active | Native Linux development |
| WASM | Wasmtime | 📋 Planned | Browser/edge execution |

## Quality Standards

### Go Code Quality

- **Formatter**: `go fmt` (mandatory)
- **Linter**: `go vet`, `golangci-lint`
- **Tests**: `go test` with coverage >70%
- **Dependencies**: Use `go mod`, no vendor/

### Test Requirements

```bash
# Unit tests
go test ./...

# Integration tests (requires platform tooling)
go test ./... -tags=integration

# Benchmark tests
go test -bench=. ./...
```

## Git Workflow

### Branch Naming

Format: `<type>/<platform>/<description>`

Types: `feat`, `fix`, `docs`, `refactor`, `test`

Examples:
- `feat/mac/lima-adapter`
- `fix/windows/wsl2-path`
- `docs/api-reference`

### Commit Messages

Format: `<type>(<platform>): <description>`

Examples:
- `feat(mac): add Lima adapter with vz driver support`
- `fix(windows): handle WSL2 path conversion on NTFS`
- `docs(linux): add gVisor integration guide`

## Documentation

- VitePress for user-facing docs
- Run `npm run docs:dev` for local preview
- Docs deploy automatically to GitHub Pages on main branch push

## CLI Commands

```bash
# Show platform info
devenv-abstraction info

# Create sandbox
devenv-abstraction create --name myapp --platform mac

# List sandboxes
devenv-abstraction list

# Delete sandbox
devenv-abstraction delete <sandbox-id>

# Pull OCI image
devenv-abstraction pull oci://localhost:5000/myapp:latest
```

## Troubleshooting

### Lima not found (macOS)
```bash
# Install Lima
brew install lima

# Verify
limactl --version
```

### WSL2 not found (Windows)
```powershell
# Install WSL2
wsl --install

# Verify
wsl --list --verbose
```

### gVisor not available
```bash
# Install gVisor
go install github.com/google/gvisor/gvisor@latest
```
