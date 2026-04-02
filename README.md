# devenv-abstraction

> Docker-alternative VM stack with OCI/sandbox support

A Go-based container runtime abstraction layer providing a unified interface for managing development environments across Mac, Windows (WSL), and Linux platforms.

## Features

- **Multi-Platform Support**: Works on macOS, Windows (WSL2), and Linux
- **OCI Compatible**: Outputs OCI-compliant bundles
- **Hexagonal Architecture**: Clean separation of concerns
- **Extensible Adapters**: Easy to add new platform backends
- **Resource Management**: CPU, memory, disk, and network limits

## Architecture

```
devenv-abstraction/
├── cmd/                    # CLI entry point
├── internal/
│   ├── adapters/           # Platform implementations
│   │   ├── mac/          # macOS (Lima + vz)
│   │   ├── windows/      # Windows (WSL2 + gVisor)
│   │   └── linux/        # Linux (gVisor + crun)
│   ├── core/              # Business logic
│   ├── domain/            # Domain models
│   └── ports/            # Interface definitions
├── pkg/                   # Shared packages
└── docs/                 # Documentation
```

## Quick Start

```bash
# Clone and build
git clone https://github.com/KooshaPari/devenv-abstraction.git
cd devenv-abstraction
go build ./cmd/devenv-abstraction

# List available sandboxes
./devenv-abstraction list

# Create a new sandbox
./devenv-abstraction create my-dev-env --platform mac

# Start a sandbox
./devenv-abstraction start my-dev-env

# Execute a command
./devenv-abstraction exec my-dev-env -- go version

# Stop a sandbox
./devenv-abstraction stop my-dev-env
```

## Platform Backends

| Platform | Backend | Requirements |
|----------|---------|-------------|
| macOS | Lima + vz | macOS 13+, Apple Silicon or Intel with VT-x |
| Windows | WSL2 + gVisor | Windows 11 with WSL2 enabled |
| Linux | gVisor + crun | Linux 5.x with kernel support |

## Configuration

Create a `devenv.toml`:

```toml
[defaults]
platform = "mac"
resources.cpu = 4
resources.memory = "8GB"
resources.disk = "50GB"

[sandboxes.my-dev-env]
platform = "mac"
resources.cpu = 8
resources.memory = "16GB"
```

## Development

```bash
# Run tests
go test ./...

# Run linter
golangci-lint run

# Build for all platforms
make build-all
```

## Documentation

Full documentation at: https://KooshaPari.github.io/devenv-abstraction

## License

MIT
