# NanoVMS Specification

> Nano Virtual Machine Services — Headless VM abstraction for AI agents

**Version**: 2.0
**Status**: Draft
**Last Updated**: 2026-04-02

## Overview

NanoVMS provides lightweight, headless virtual machine services for AI agent-driven development workflows. It implements a **three-tier isolation architecture** optimized for ephemeral workloads:

1. **Tier 1: WASM Sandboxes** — Trusted tools, fast execution
2. **Tier 2: gVisor Containers** — Semi-trusted, syscall filtering
3. **Tier 3: MicroVMs** — Untrusted code, full hardware isolation

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Agent Controller                               │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 1: WASM Sandboxes                                              │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐               │
│  │ Tool A (.wasm)│ │ Tool B (.wasm)│ │ Tool C (.wasm)│               │
│  └──────────────┘ └──────────────┘ └──────────────┘               │
│  ├── Fast startup (~1ms)                                             │
│  ├── Memory limits enforced                                           │
│  └── No direct syscalls (WASI sandbox)                               │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 2: gVisor Containers                                           │
│  ┌──────────────────────────────────────────────┐                   │
│  │  Sentry (userspace kernel)                    │                   │
│  │  └── Syscall interception + filtering         │                   │
│  └──────────────────────────────────────────────┘                   │
│  ├── Network isolation                                               │
│  ├── Filesystem filtering                                            │
│  └── ~90ms startup                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 3: MicroVMs (Firecracker/Cloud Hypervisor)                    │
│  ┌──────────────────────────────────────────────┐                   │
│  │  Firecracker MicroVM                         │                   │
│  │  ├── VT-x/AMD-V isolation                    │                   │
│  │  ├── Minimal device model (5 devices)        │                   │
│  │  └── ~125ms startup, <5MB RAM               │                   │
│  └──────────────────────────────────────────────┘                   │
│  ├── Full OCI image compatibility                                    │
│  └── Untrusted code execution                                        │
└─────────────────────────────────────────────────────────────────────┘
```

## Two-Level Abstraction

### Level 1: Infrastructure Layer (CURRENT)

VM runtime implementations:

| Adapter | Platform | Technology | Status |
|---------|----------|------------|--------|
| `mac/` | macOS | Lima/VZ + virtiofs | ✅ Implemented |
| `windows/` | Windows | WSL2 + gVisor | ⚠️ Partial |
| `linux/` | Linux | Native/KVM | ⚠️ Partial |
| `microvm/` | All | Firecracker | 📋 Planned |
| `wasm/` | All | Wasmtime | ⚠️ Partial |
| `sandbox/` | Linux | bwrap, firejail | ✅ Implemented |

### Level 2: Platform Target Layer (ROADMAP)

Development environment targets:

| Category | Targets | Infrastructure | Priority |
|----------|---------|----------------|----------|
| Apple | iOS, iPadOS, tvOS, watchOS, visionOS | Lima/VZ | P2 |
| Android | Phone, Tablet, Wear, TV, Auto | WSL2/Lima | P2 |
| Smart TV | Tizen, webOS, Roku, Fire TV | QEMU | P3 |
| Gaming | PlayStation, Switch, Xbox | Various | P3 |
| AR/VR | Quest, HoloLens, SteamVR, SteamOS | Remote/QEMU | P3 |
| IoT | Raspberry Pi, Pine64, ESP32 | QEMU | P3 |

## VM Tiers

| Tier | Technology | Startup | Memory | Use Case |
|------|------------|---------|--------|----------|
| **1 - WASM** | Wasmtime | ~1ms | ~1MB | Tool execution |
| **2 - gVisor** | runsc | ~90ms | ~20MB | Semi-trusted |
| **3 - MicroVM** | Firecracker | ~125ms | <5MB | Untrusted |
| **4 - Native** | KVM/Hyper-V | ~1s | Varies | Production |

## Core Types

### VMFlavor — Infrastructure Layer

```go
type VMFlavor string

const (
    VMFlavorNative  VMFlavor = "native"   // KVM, Hyper-V, HyperKit
    VMFlavorLima    VMFlavor = "lima"     // Lima with vz driver (macOS)
    VMFlavorWSL     VMFlavor = "wsl"      // Windows Subsystem for Linux
    VMFlavorMicroVM VMFlavor = "microvm"  // Firecracker, Cloud Hypervisor
    VMFlavorWasm    VMFlavor = "wasm"     // WebAssembly runtime
)
```

### SandboxType — Isolation Tiers

```go
type SandboxType string

const (
    SandboxTypeWasm      SandboxType = "wasm"      // Tier 1: Fast sandbox
    SandboxTypeProcess   SandboxType = "process"   // Tier 2: gVisor, landlock
    SandboxTypeContainer SandboxType = "container" // Tier 2: Container + gVisor
    SandboxTypeNative    SandboxType = "native"    // Tier 3: bwrap, firejail
    SandboxTypeVM        SandboxType = "vm"        // Tier 3: Full VM
)
```

### IsolationTier — Decision Framework

```go
type IsolationTier int

const (
    Tier1Wasm      IsolationTier = 1 // Trusted tools, fast iteration
    Tier2Gvisor    IsolationTier = 2 // Semi-trusted, syscall filtering
    Tier3MicroVM   IsolationTier = 3 // Untrusted, full isolation
)

// Tier selection based on trust level:
// Tier1Wasm:    Agent-native tools, first-party code
// Tier2Gvisor:  Third-party tools, scripts with network access
// Tier3MicroVM: LLM-generated code, untrusted executables
```

## Performance Benchmarks

| Component | Technology | Startup | Memory | Cold Start |
|-----------|------------|---------|--------|------------|
| WASM Tool | Wasmtime | ~1ms | ~1MB | ~1ms |
| gVisor Container | runsc | ~90ms | ~20MB | ~90ms |
| Firecracker MicroVM | Rust | ~125ms | <5MB | ~125ms |
| runc Container | Go | ~1s | ~50MB | ~500ms |
| Docker Container | Go | ~1s+ | ~100MB | ~1s |

## Security Model

| Tier | Isolation | Attack Surface | Compliance |
|------|-----------|----------------|------------|
| 1 - WASM | Language sandbox | Minimal (WASI only) | High |
| 2 - gVisor | Userspace kernel | Low (syscall filter) | High |
| 3 - MicroVM | Hardware VT-x/AMD-V | Minimal (5 devices) | Highest |

### Sandbox Isolation Implementations

| Implementation | Type | Platform | Status |
|----------------|------|----------|--------|
| bwrap | Native | Linux | ✅ Implemented |
| firejail | Native | Linux | ✅ Implemented |
| unshare | Native | Linux | ✅ Implemented |
| gVisor (runsc) | Process | Linux | 📋 Planned |
| landlock | Process | Linux 5.13+ | 📋 Planned |
| seccomp | Process | Linux | 📋 Planned |
| sandbox-exec | Native | macOS | ⚠️ Partial |

## Headless IDE Support

### VS Code Server

```bash
# Headless VS Code for agents
nanovms ide start --type vscode-server --workspace /project
```

### JetBrains Gateway

```bash
# JetBrains Projector backend
nanovms ide start --type jetbrains --project /project
```

### Development Containers

```bash
# Dev container with tools pre-installed
nanovms devcontainer create --image ubuntu:22.04-dev
```

## Mobile Simulator Support

### iOS Simulators

| Platform | Simulator | CLI Tool | Headless |
|----------|-----------|----------|----------|
| iOS | iPhone, iPad | `xcrun simctl` | ⚠️ Limited |
| iPadOS | iPad | `xcrun simctl` | ⚠️ Limited |
| tvOS | Apple TV | `xcrun simctl` | ⚠️ Limited |
| watchOS | Apple Watch | `xcrun simctl` | ⚠️ Limited |
| visionOS | Vision Pro | `xcrun simctl` | ⚠️ Limited |

### Android Emulators

| Platform | Emulator | Headless | CI Support |
|----------|----------|----------|------------|
| Phone | AVD | ✅ | `emulator -no-window` |
| Tablet | AVD | ✅ | `emulator -no-window` |
| TV | AVD (Leanback) | ✅ | `emulator -no-window` |
| Wear | AVD | ⚠️ | Remote stream |
| Auto | AVD | ✅ | `emulator -no-window` |

### AR/VR Simulators

| Platform | Simulator | Hypervisor | Notes |
|----------|-----------|------------|-------|
| visionOS | Xcode | Lima/VZ | Vision Pro only |
| Meta Quest | Meta Horizon | Remote | Stream to device |
| HoloLens | HoloLens Emulator | Hyper-V | Windows only |
| SteamVR | Steam | Proton/Wine | Windows |
| SteamOS | ChimeraOS | QEMU | Steam Deck |

## Minimal Init Systems

For ephemeral workloads, NanoVMS uses minimal init systems:

| Init | Boot Time | Memory | Use Case |
|------|-----------|--------|----------|
| **runit** | ~100ms | ~1MB | Ephemeral containers |
| **OpenRC** | ~500ms | ~5MB | Alpine-based images |
| **s6** | ~50ms | ~500KB | Multi-agent orchestration |
| **systemd** | ~2s | ~50MB | Full-featured VMs |

### Fast Boot Configuration

```dockerfile
FROM alpine:3.19
RUN apk add --no-cache runit
# Disable unnecessary services
RUN find /etc/runlevels -type l -delete 2>/dev/null || true
```

## Quality Gates

```bash
go fmt ./...        # Format
go vet ./...        # Vet
go build ./...      # Build
go test ./...       # Tests
golangci-lint run   # Lint
```

## Status Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Implemented and tested |
| ⚠️ | Partial implementation (stub) |
| 📋 | Planned (not started) |
| ❌ | Not supported |

## References

- [Firecracker](https://github.com/firecracker-microvm/firecracker) — AWS microVM
- [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor) — Intel VMM
- [Wasmtime](https://github.com/bytecodealliance/wasmtime) — Bytecode Alliance WASM
- [gVisor](https://github.com/google/gvisor) — Google sandbox
- [Lima](https://github.com/lima-vm/lima) — Linux VM on macOS
- [runit](http://smarden.org/runit/) — Process supervision

---

*This spec reflects NanoVMS v2.0 architecture based on 2026 research.*
