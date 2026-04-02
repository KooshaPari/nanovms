# Devenv Abstraction

A Docker-alternative VM stack with OCI/sandbox support. Provides a unified interface for managing development environments across Mac, Windows (WSL), and Linux.

## Features

- **Multi-Platform Support**: Mac native (Lima/Colima), Windows (WSL2), Linux native
- **OCI Compliant**: Uses OCI runtime specifications for container management
- **Hexagonal Architecture**: Clean separation of concerns with ports and adapters
- **Sandbox Isolation**: Supports gVisor, namespace isolation, and MicroVMs

## Quick Start

```bash
# Clone the repository
git clone https://github.com/KooshaPari/devenv-abstraction.git
cd devenv-abstraction

# Build
go build ./cmd/devenv-abstraction

# Run
./devenv-abstraction --help
```

## Architecture

Devenv Abstraction uses a hexagonal (ports and adapters) architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Core                        │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                 Domain (Sandbox)                      │  │
│  │  - Sandbox entity                                    │  │
│  │  - Lifecycle management                               │  │
│  │  - Configuration                                     │  │
│  └─────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                   Ports (Interfaces)                 │  │
│  │  - RuntimePort                                      │  │
│  │  - FilesystemPort                                   │  │
│  │  - NetworkPort                                      │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│   Mac Adapter │ │ Windows Adapt │ │ Linux Adapter │
│  (Lima/vz)   │ │   (WSL2/gVis)│ │  (gVisor)    │
└───────────────┘ └───────────────┘ └───────────────┘
```

## Platform Support

| Platform | Primary Runtime | Isolation | Status |
|----------|----------------|-----------|--------|
| macOS | Lima/Colima + vz | Namespace | Stable |
| Windows | WSL2 + gVisor | Syscall interception | Stable |
| Linux | gVisor/crun | Syscall filtering | Stable |

## License

MIT
