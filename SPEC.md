# NanoVMS Specification

> Nano Virtual Machine Services — Headless VM abstraction for agents

## Overview

NanoVMS provides lightweight, headless virtual machine services for agent-driven development workflows. It implements a **two-level abstraction**:

1. **Infrastructure Layer** (CURRENT): VM runtimes (Native, Lima, WSL, MicroVM, WASM)
2. **Platform Layer** (PLANNED): Target platforms (iOS, Android, tvOS, etc.)

## Architecture

```
nanovms/
├── internal/
│   ├── domain/           # Core models (Sandbox, VMConfig, VMFlavor)
│   ├── ports/           # Port interfaces (VMAdapter, SandboxManager)
│   ├── adapters/         # VM runtime adapters
│   │   ├── mac/         # Lima/VZ adapter
│   │   ├── windows/     # WSL adapter
│   │   ├── linux/       # Native/KVM adapter
│   │   ├── microvm/     # Firecracker adapter
│   │   ├── wasm/        # WASM runtime adapter
│   │   └── sandbox/     # Sandbox layer adapters (bwrap, gVisor, etc.)
│   └── core/            # Orchestration logic
├── cmd/nanovms/         # CLI entry point
├── pkg/                 # Public API library
└── SPEC.md
```

## Core Types (IMPLEMENTED)

### VMFlavor - Infrastructure Layer

```go
// internal/domain/sandbox.go

type VMFlavor string

const (
    VMFlavorNative  VMFlavor = "native"   // HyperKit, Hyper-V, KVM
    VMFlavorLima    VMFlavor = "lima"     // Lima with vz driver (macOS)
    VMFlavorWSL     VMFlavor = "wsl"      // Windows Subsystem for Linux
    VMFlavorMicroVM VMFlavor = "microvm"  // Firecracker microVM
    VMFlavorWasm    VMFlavor = "wasm"     // WebAssembly runtime
)
```

### SandboxType - Isolation Levels

```go
type SandboxType string

const (
    SandboxTypeVM        SandboxType = "vm"        // Full virtual machine
    SandboxTypeContainer SandboxType = "container" // Container isolation
    SandboxTypeWasm      SandboxType = "wasm"      // WebAssembly isolation
    SandboxTypeProcess   SandboxType = "process"   // gVisor, landlock
    SandboxTypeNative    SandboxType = "native"     // bwrap, firejail, namespaces
)
```

### NativeSandboxType - Linux Sandbox Implementations

```go
type NativeSandboxType string

const (
    NativeSandboxBwrap           NativeSandboxType = "bwrap"
    NativeSandboxFirejail        NativeSandboxType = "firejail"
    NativeSandboxUnshare         NativeSandboxType = "unshare"
    NativeSandboxChroot          NativeSandboxType = "chroot"
    NativeSandboxWindowsContainer NativeSandboxType = "windows-container"
    NativeSandboxMacOSContain     NativeSandboxType = "sandbox-exec"
)
```

## VM Tiers

| Tier | Technology | Use Case | Overhead |
|------|------------|----------|----------|
| **1 - Native** | HyperKit, Hyper-V, KVM | Production parity | Highest |
| **2 - Lima/WSL** | Lima/VZ, WSL2 | Daily development | Medium |
| **3 - MicroVM** | Firecracker, Cloud Hypervisor | CI/CD, isolation | Lowest |
| **4 - WASM** | Wasmtime, Wasmer | Lightweight execution | Minimal |

## Security Model

| Tier | Isolation | Performance | Use Case |
|------|-----------|-------------|----------|
| Native VM | VT-x/AMD-V | ★★★★☆ | Production testing |
| MicroVM | Firecracker | ★★★★★ | Agent sandboxing |
| Container | namespaces/cgroups | ★★★★★ | Local dev |
| WASM | Bytecode isolation | ★★★★★ | Lightweight workloads |

Sandbox layers (gVisor, landlock, seccomp, WASM) can be stacked for additional security.

## Platform Support Matrix (CURRENT)

### Infrastructure Adapters

| Adapter | Platform | Status | Notes |
|---------|----------|--------|-------|
| `mac/` | macOS | ✅ Implemented | Lima with vz driver |
| `windows/` | Windows | ⚠️ Partial | WSL2 adapter (stub) |
| `linux/` | Linux | ⚠️ Partial | Native/KVM (stub) |
| `microvm/` | All | 📋 Planned | Firecracker adapter |
| `wasm/` | All | ⚠️ Partial | WASM adapter (stub) |
| `sandbox/` | Linux | ✅ Implemented | bwrap, firejail |

### Host Requirements

| VMFlavor | macOS | Windows | Linux |
|----------|-------|---------|-------|
| Native | HyperKit | Hyper-V | KVM |
| Lima/WSL | ✅ Lima + VZ | ✅ WSL2 | N/A |
| MicroVM | ✅ Firecracker | ✅ Firecracker | ✅ Firecracker |
| WASM | ✅ Wasmtime | ✅ Wasmtime | ✅ Wasmtime |

## Planned: Platform Target Layer (ROADMAP)

The second abstraction layer maps infrastructure flavors to platform targets:

### Apple Platforms

| Target | Simulator | Infrastructure | Status |
|--------|-----------|----------------|--------|
| iOS | iPhone, iPad | Lima/VZ | 📋 Planned |
| iPadOS | iPad | Lima/VZ | 📋 Planned |
| tvOS | Apple TV | Lima/VZ | 📋 Planned |
| watchOS | Apple Watch | Lima/VZ | 📋 Planned |
| visionOS | Vision Pro | Lima/VZ | 📋 Planned |
| macOS | N/A | Native/Lima | ⚠️ Partial |

### Android Ecosystem

| Target | Emulator | Infrastructure | Status |
|--------|----------|----------------|--------|
| Phone | Android Emulator | WSL2/Lima | 📋 Planned |
| Tablet | Various | WSL2/Lima | 📋 Planned |
| Wear OS | Wear device | Remote stream | 📋 Planned |
| Android TV | TV emulator | WSL2/Lima | 📋 Planned |
| Automotive | Auto emulator | WSL2/Lima | 📋 Planned |

### Smart TV Platforms

| Target | SDK/Emulator | Infrastructure | Status |
|--------|--------------|----------------|--------|
| tvOS | Xcode | Lima/VZ | 📋 Planned |
| Android TV | Android Emulator | WSL2/Lima | 📋 Planned |
| Samsung Tizen | Tizen Studio | QEMU | 📋 Planned |
| LG webOS | webOS SDK | QEMU | 📋 Planned |
| Roku | Roku OS SDK | QEMU | 📋 Planned |
| Fire TV | Fire OS Emulator | WSL2 | 📋 Planned |

### Gaming Consoles

| Target | Simulator | Infrastructure | Status |
|--------|-----------|----------------|--------|
| PlayStation | PS DevKit | Remote only | 📋 Planned |
| Nintendo Switch | Yuzu/Ryujinx | Lima + Wine | 📋 Planned |
| Xbox | Dev Mode | Hyper-V | 📋 Planned |

### AR/VR Platforms

| Target | Runtime | Infrastructure | Status |
|--------|---------|----------------|--------|
| visionOS | Xcode | Lima/VZ | 📋 Planned |
| Meta Quest | Meta Horizon | Remote stream | 📋 Planned |
| HoloLens | HoloLens Emulator | Hyper-V | 📋 Planned |
| Magic Leap | Magic Leap Lab | Remote | 📋 Planned |
| SteamVR | SteamVR | Proton/Wine | 📋 Planned |
| SteamOS | ChimeraOS | QEMU | 📋 Planned |

### IoT / Embedded

| Target | Emulator | Infrastructure | Status |
|--------|----------|----------------|--------|
| Raspberry Pi | QEMU | ARM emulation | 📋 Planned |
| Pine64 | QEMU | ARM64 emulation | 📋 Planned |
| ESP32/STM32 | QEMU | Embedded targets | 📋 Planned |
| FreeRTOS | QEMU | RTOS targets | 📋 Planned |

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

---

*This spec reflects the current implementation state. Platform target support is planned for future releases.*
