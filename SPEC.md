# devenv-abstraction Specification

> Docker-alternative VM stack with OCI/sandbox support - 3-tier architecture

## Overview

`devenv-abstraction` is a Go-based container runtime abstraction layer that provides a unified interface for managing development environments across Mac, Windows, and Linux platforms with three isolation tiers plus sandboxing.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          CLI / API                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                       Ports (Interfaces)                                │
│  ┌──────────────┐ ┌──────────────┐ ┌────────────┐ ┌────────────┐  │
│  │ RuntimePort   │ │ SandboxPort  │ │VMAdapterPort│ │WASMAdapter  │  │
│  │              │ │              │ │            │ │ Port       │  │
│  └──────────────┘ └──────────────┘ └────────────┘ └────────────┘  │
├─────────────────────────────────────────────────────────────────────────┤
│                       Core (Business Logic)                            │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐     │
│  │ Sandbox      │ │ Lifecycle    │ │ Resource Manager         │     │
│  │ Orchestrator │ │ Manager      │ │                         │     │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘     │
├─────────────────────────────────────────────────────────────────────────┤
│                    Sandbox Isolation Layer                             │
│  ┌──────────────┐ ┌──────────────┐ ┌────────────┐ ┌────────────┐    │
│  │ gVisor       │ │ landlock     │ │ seccomp    │ │ wasmtime   │    │
│  │ (runsc)      │ │ (Linux)      │ │ (all)      │ │ (WASM)     │    │
│  └──────────────┘ └──────────────┘ └────────────┘ └────────────┘    │
├─────────────────────────────────────────────────────────────────────────┤
│                    VM Adapter Layer (3 Tiers)                          │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │ TIER 1: Native VM (Full isolation, highest overhead)             │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐│  │
│  │  │ Mac:         │ │ Windows:    │ │ Linux:                  ││  │
│  │  │ HyperKit /   │ │ Hyper-V /   │ │ KVM / QEMU             ││  │
│  │  │ Virt Framework│ │ VMware      │ │                         ││  │
│  │  └──────────────┘ └──────────────┘ └──────────────────────────┘│  │
│  ├─────────────────────────────────────────────────────────────────┤  │
│  │ TIER 2: Container/WSL (OS-level, medium overhead)               │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐│  │
│  │  │ Mac:         │ │ Windows:    │ │ Linux:                  ││  │
│  │  │ Lima + VZ    │ │ WSL2        │ │ Native namespaces       ││  │
│  │  │ driver       │ │             │ │ + cgroups v2           ││  │
│  │  └──────────────┘ └──────────────┘ └──────────────────────────┘│  │
│  ├─────────────────────────────────────────────────────────────────┤  │
│  │ TIER 3: MicroVM (Lightweight VM, low overhead)                  │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐│  │
│  │  │ Firecracker  │ │ Firecracker  │ │ Firecracker /           ││  │
│  │  │ (AWS)        │ │ (Cross-OS)   │ │ Cloud Hypervisor        ││  │
│  │  └──────────────┘ └──────────────┘ └──────────────────────────┘│  │
│  └─────────────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────────┤
│                       OCI Runtime Layer                                │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────────┐   │
│  │ runc         │ │ gVisor      │ │ youki / crun               │   │
│  │ (default)    │ │ (sandboxed) │ │ (OCI compliant)            │   │
│  └──────────────┘ └──────────────┘ └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Platform Support

| Platform | Tier 1 (Native VM) | Tier 2 (Container/WSL) | Tier 3 (MicroVM) |
|----------|-------------------|----------------------|------------------|
| **macOS** | ✅ Virt.framework (Apple Silicon) / HyperKit (Intel) | ✅ Lima + vz driver | ✅ Firecracker |
| **Windows** | ✅ Hyper-V / VMware Workstation | ✅ WSL2 | ✅ Firecracker (via WSL2) |
| **Linux** | ✅ KVM / QEMU | ✅ Native namespaces + cgroups v2 | ✅ Firecracker / Cloud Hypervisor |

## Sandbox Isolation Support

| Sandbox Type | Description | Platforms | Use Case |
|-------------|-------------|----------|----------|
| **Native Sandboxes** | Lightweight process isolation | | |
| `bwrap` (bubblewrap) | Linux namespace sandbox, no VM | Linux | Fast dev envs, CI |
| `firejail` | AppArmor/sandbox profiles | Linux | GUI apps, network isolation |
| `unshare` | Manual namespace creation | Linux | Minimal containers |
| **Native macOS** | macOS-specific sandboxing | | |
| `sandbox-exec` | Apple sandbox profiles | macOS | App Store compliance |
| **Native Windows** | Windows-specific sandboxing | | |
| `Windows Sandbox` | Hyper-V lightweight VM | Windows | Quick testing |
| **Kernel Sandboxes** | Syscall interception | | |
| **gVisor** | User-space kernel, syscall interception | All (via runc) | High-security isolation |
| **landlock** | Filesystem sandboxing | Linux 5.13+ | Filesystem restrictions |
| **seccomp** | Syscall filtering | All | System call filtering |
| **WASM** | WebAssembly runtime | All | Language-level isolation |

### Native Sandbox Performance

| Sandbox | Spin-up Time | Memory Overhead | Isolation Level |
|---------|-------------|----------------|-----------------|
| `bwrap` | < 10ms | ~0 | Process namespace |
| `firejail` | < 50ms | ~1MB | AppArmor + namespaces |
| `unshare` | < 5ms | ~0 | Namespaces only |
| `sandbox-exec` | < 20ms | ~0 | Seatbelt profiles |
| gVisor | ~100ms | ~50MB | User-space kernel |
| MicroVM (Firecracker) | ~100ms | ~5MB | Full VM |

## Core Types

### Sandbox

```go
type Sandbox struct {
    ID        string
    Name      string
    Status    SandboxStatus
    Platform  Platform
    VMTier    VMTier
    Sandbox   SandboxType
    Config    SandboxConfig
}
```

### VMTier

```go
// VMTier represents the VM isolation tier
type VMTier int

const (
    Tier1NativeVM VMTier = iota + 1 // HyperKit, Hyper-V, KVM
    Tier2Container                   // Lima, WSL2, namespaces
    Tier3MicroVM                     // Firecracker, Cloud Hypervisor
)
```

### SandboxType

```go
// SandboxType represents the sandbox isolation type
type SandboxType int

const (
    SandboxNone SandboxType = iota
    SandboxGVisor    // gVisor user-space kernel
    SandboxLandlock  // Linux landlock filesystem sandbox
    SandboxSeccomp  // seccomp syscall filtering
    SandboxWASM     // WASM runtime isolation
)
```

### Platform

```go
type Platform string

const (
    PlatformMac     Platform = "mac"
    PlatformWindows Platform = "windows"
    PlatformLinux   Platform = "linux"
)
```

## Ports (Interfaces)
## Adapter Interfaces

### VMAdapter (Port)

```go
// VMAdapter is implemented by platform-specific VM adapters
type VMAdapter interface {
    // Platform returns the platform this adapter supports
    Platform() Platform

    // VMTier returns the tier this adapter implements
    VMTier() VMTier

    // Start starts the VM
    Start(ctx context.Context, config VMConfig) error

    // Stop stops the VM
    Stop(ctx context.Context) error

    // Status returns the current VM status
    Status(ctx context.Context) (VMStatus, error)

    // Exec executes a command inside the VM
    Exec(ctx context.Context, cmd []string, opts ExecOptions) error

    // Mount mounts a directory into the VM
    Mount(ctx context.Context, source, target string, readonly bool) error
}
```

### SandboxPort

```go
// SandboxPort is implemented by sandbox adapters
type SandboxPort interface {
    // SandboxType returns the type of sandbox this adapter implements
    SandboxType() SandboxType

    // Apply applies the sandbox profile to a process
    Apply(ctx context.Context, pid int, profile SandboxProfile) error

    // Validate validates a sandbox profile
    Validate(profile SandboxProfile) error
}
```

### WASMAdapterPort

```go
// WASMAdapterPort is implemented by WASM runtime adapters
type WASMAdapterPort interface {
    // Runtime returns the WASM runtime name
    Runtime() string

    // Compile compiles WASM module
    Compile(ctx context.Context, wasm []byte) (CompiledModule, error)

    // Instantiate creates a new WASM instance
    Instantiate(ctx context.Context, module CompiledModule, imports WASMImports) error
}
```
## Sandbox Lifecycle

```
create ──► pending ──► running ──► stopped
              │           │
              │           ▼
              └─────────► error
              │
              ▼
           deleted
```

## OCI Compliance

The runtime adapter outputs OCI-compatible bundles when possible, ensuring compatibility with existing container tooling.

## Resource Limits

| Resource | Default | Max |
|----------|---------|-----|
| CPU | 2 cores | 8 cores |
| Memory | 4GB | 32GB |
| Disk | 20GB | 100GB |
| Networks | 1 | 4 |

## Error Handling

All errors wrap the underlying system error with context:

```go
var ErrSandboxNotFound = errors.New("sandbox not found")
var ErrRuntimeUnavailable = errors.New("runtime unavailable")
var ErrInvalidConfig = errors.New("invalid sandbox configuration")
```

## Security Model

- **Mac**: Lima VMs are isolated via hypervisor.framework (Apple Silicon) or HVF (Intel)
- **Windows**: WSL2 provides kernel isolation; gVisor adds syscall filtering
- **Linux**: Native namespace isolation with gVisor syscall interception

## File Format

Sandbox configurations are stored in TOML:

```toml
name = "my-dev-env"
platform = "mac"
resources.cpu = 4
resources.memory = "8GB"
resources.disk = "50GB"
```
