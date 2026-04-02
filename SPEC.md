# devenv-abstraction Specification

> Docker-alternative VM stack with OCI/sandbox support

## Overview

`devenv-abstraction` is a Go-based container runtime abstraction layer that provides a unified interface for managing development environments across Mac, Windows (WSL), and Linux platforms.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI / API                                 │
├─────────────────────────────────────────────────────────────────┤
│                      Ports (Interfaces)                         │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │RuntimePort   │ │ SandboxPort  │ │ ProcessPort              │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Core (Business Logic)                       │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │Sandbox       │ │ Lifecycle    │ │ Resource Manager          │ │
│  │Orchestrator  │ │ Manager      │ │                          │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Adapters (Implementations)                 │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │ Mac          │ │ Windows/WSL  │ │ Linux Native             │ │
│  │ (Lima/VZ)   │ │ (WSL2+gVisor│ │ (gVisor/crun)           │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      OCI Runtime Layer                           │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │ runc         │ │ gVisor      │ │ youki / crun             │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Platform Support

| Platform | Backend | Status | Isolation |
|----------|---------|--------|-----------|
| macOS | Lima + vz driver | ✅ Stable | Namespace + cgroup |
| Windows | WSL2 + gVisor | ✅ Stable | Syscall interception |
| Linux | gVisor + crun | ✅ Stable | Ptrace sandboxing |

## Core Types

### Sandbox

```go
type Sandbox struct {
    ID       string
    Name     string
    Status   SandboxStatus
    Platform Platform
    Config   SandboxConfig
}
```

### SandboxStatus

```go
type SandboxStatus string

const (
    StatusPending   SandboxStatus = "pending"
    StatusRunning   SandboxStatus = "running"
    StatusStopped   SandboxStatus = "stopped"
    StatusError     SandboxStatus = "error"
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

### RuntimePort

```go
type RuntimePort interface {
    // Name returns the adapter name
    Name() string

    // IsAvailable checks if the runtime is available on this system
    IsAvailable() bool

    // Create creates a new sandbox
    Create(ctx context.Context, name string) (string, error)

    // Delete removes a sandbox
    Delete(ctx context.Context, id string) error

    // List returns all sandboxes
    List(ctx context.Context) ([]Sandbox, error)

    // Start starts a sandbox
    Start(ctx context.Context, id string) error

    // Stop stops a sandbox
    Stop(ctx context.Context, id string) error

    // Exec executes a command in a sandbox
    Exec(ctx context.Context, id string, cmd []string) ([]byte, error)
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
