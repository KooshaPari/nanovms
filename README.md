# NanoVMS - Nano Virtual Machine Services

@trace VM-001: MicroVM Lifecycle
@trace VM-002: WASM Sandbox
@trace VM-003: Container Isolation
@trace VM-004: Firecracker Integration
@trace VM-009: Snapshot Management
@trace VM-010: Storage Backend
@trace VM-011: Networking (Cilium)
@trace VM-012: Security Model
@trace VM-013: CLI Interface
@trace VM-014: Multi-Platform

> Lightweight, headless VM abstraction for AI agents — Three-tier isolation architecture

NanoVMS provides a unified interface for managing development environments and isolated execution across desktop, mobile, embedded, gaming, and emerging platforms.

## Three-Tier Isolation Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Agent Controller                               │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 1: WASM Sandboxes (~1ms startup, ~1MB memory)              │
│  └── Fast tool execution, WASI sandbox, no syscalls                │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 2: gVisor Containers (~90ms startup, ~20MB memory)           │
│  └── Syscall filtering, network isolation                           │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 3: MicroVMs (~125ms startup, <5MB memory)                    │
│  └── Firecracker, OCI compatible, full hardware isolation           │
└─────────────────────────────────────────────────────────────────────┘
```

## Features

### Isolation Tiers

| Tier | Technology | Startup | Memory | Use Case |
|------|------------|---------|--------|----------|
| **1** | Wasmtime | ~1ms | ~1MB | Agent tools, plugins |
| **2** | gVisor (runsc) | ~90ms | ~20MB | Semi-trusted code |
| **3** | Firecracker | ~125ms | <5MB | Untrusted code |

### Infrastructure Adapters

| Adapter | Platform | Technology | Status |
|---------|----------|------------|--------|
| `mac/` | macOS | Lima/VZ + virtiofs | ✅ Implemented |
| `windows/` | Windows | WSL2 + gVisor | ⚠️ Partial |
| `linux/` | Linux | Native/KVM | ⚠️ Partial |
| `microvm/` | All | Firecracker | 📋 Planned |
| `wasm/` | All | Wasmtime | ⚠️ Partial |
| `sandbox/` | Linux | bwrap, firejail | ✅ Implemented |

### Platform Targets (ROADMAP)

| Category | Targets | Infrastructure |
|----------|---------|----------------|
| Apple | iOS, iPadOS, tvOS, watchOS, visionOS | Lima/VZ |
| Android | Phone, Tablet, Wear, TV, Auto | WSL2/Lima |
| Smart TV | Tizen, webOS, Roku, Fire TV | QEMU |
| Gaming | PlayStation, Switch, Xbox | Various |
| AR/VR | Quest, HoloLens, SteamVR, SteamOS | Remote/QEMU |
| IoT | Raspberry Pi, Pine64, ESP32 | QEMU |

## Architecture

```
nanovms/
├── cmd/nanovms/              # CLI entry point
├── internal/
│   ├── adapters/             # Infrastructure implementations
│   │   ├── mac/             # Lima/VZ adapter (macOS)
│   │   ├── windows/         # WSL adapter (Windows)
│   │   ├── linux/           # Native/KVM adapter (Linux)
│   │   ├── microvm/         # Firecracker adapter (all platforms)
│   │   ├── wasm/            # Wasmtime adapter
│   │   └── sandbox/         # bwrap, firejail, gVisor
│   ├── domain/               # Core models (Sandbox, VMFlavor)
│   └── ports/                # Interface definitions (VMAdapter)
└── pkg/                      # Public API library
```

## Quick Start

```bash
# Clone and build
git clone https://github.com/KooshaPari/nanovms.git
cd nanovms
go build ./cmd/nanovms

# Probe system capabilities
./nanovms probe

# Create a sandbox with specified isolation tier
./nanovms sandbox create dev --tier wasm      # Tier 1: Fast (~1ms)
./nanovms sandbox create dev --tier gvisor    # Tier 2: Secure (~90ms)
./nanovms sandbox create dev --tier microvm   # Tier 3: Full isolation (~125ms)

# Execute in sandbox
./nanovms exec dev -- node -e "console.log('hello')"
```

## CLI Commands

### Sandbox Commands

```bash
# Create sandboxes at different tiers
nanovms sandbox create <name> --tier wasm      # Tier 1: Wasmtime
nanovms sandbox create <name> --tier gvisor    # Tier 2: gVisor
nanovms sandbox create <name> --tier native    # Tier 3: bwrap/firejail

# List sandboxes
nanovms sandbox list

# Execute in sandbox
nanovms sandbox exec <name> -- <command>

# Delete sandbox
nanovms sandbox delete <name>
```

### VM Commands

```bash
# Create VM with specified flavor
nanovms vm create dev --flavor lima        # macOS: Lima/VZ
nanovms vm create dev --flavor wsl         # Windows: WSL2
nanovms vm create dev --flavor native      # Linux: KVM
nanovms vm create dev --flavor microvm     # All: Firecracker

# List VMs
nanovms vm list

# Start/stop/delete
nanovms vm start <name>
nanovms vm stop <name>
nanovms vm delete <name>
```

### IDE Commands

```bash
# Headless IDE for agents
nanovms ide start --type vscode-server --workspace /project
nanovms ide start --type jetbrains --project /project

# Dev containers
nanovms devcontainer create --image ubuntu:22.04-dev
```

## Performance

| Component | Startup | Memory | Cold Start |
|-----------|---------|--------|------------|
| WASM (Wasmtime) | ~1ms | ~1MB | ~1ms |
| gVisor Container | ~90ms | ~20MB | ~90ms |
| Firecracker MicroVM | ~125ms | <5MB | ~125ms |
| runc Container | ~1s | ~50MB | ~500ms |

## Security

| Tier | Isolation | Attack Surface |
|------|-----------|----------------|
| 1 - WASM | Language sandbox | Minimal (WASI only) |
| 2 - gVisor | Userspace kernel | Low (syscall filter) |
| 3 - MicroVM | Hardware VT-x/AMD-V | Minimal (5 devices) |

## Development

```bash
# Format
go fmt ./...

# Vet
go vet ./...

# Build
go build -o bin/nanovms ./cmd/nanovms

# Test
go test ./...

# Lint
golangci-lint run
```

## Documentation

- [SPEC.md](./SPEC.md) — Full specification
- [docs/adr/](./docs/adr/) — Architecture decision records

## License

MIT
