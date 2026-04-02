# NanoVMS - Nano Virtual Machine Services

> Lightweight, headless VM abstraction for agents — supports desktop, mobile, embedded, gaming, and emerging form factors

NanoVMS provides a unified interface for managing development environments and simulators across **desktop**, **mobile**, **embedded**, **gaming**, and **emerging platforms**.

## Features

- **Two-Level Abstraction**: Infrastructure layer (Native, Lima, WSL, MicroVM, WASM) + Platform target layer
- **Apple Ecosystem**: iOS, iPadOS, tvOS, watchOS, visionOS, macOS
- **Android Ecosystem**: Phone, Tablet, Wear OS, Android TV, Automotive
- **Smart TVs**: tvOS, Android TV, Samsung Tizen, LG webOS, Roku, Fire TV
- **Gaming**: PlayStation, Nintendo Switch, Xbox development
- **IoT/Embedded**: Raspberry Pi, Pine64, ESP32, FreeRTOS
- **AR/VR**: Meta Quest, Apple Vision Pro, Microsoft HoloLens, SteamVR, SteamOS
- **Multi-Tier VM**: Native → Lima/WSL → MicroVM (Firecracker) → WASM
- **Sandbox Isolation**: gVisor, landlock, seccomp, bwrap, WASM
- **Hexagonal Architecture**: Clean separation, extensible adapters

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
│   │   ├── wasm/            # WASM runtime adapter
│   │   └── sandbox/         # Isolation adapters (bwrap, gVisor, etc.)
│   ├── domain/               # Core models (Sandbox, VMFlavor, VMConfig)
│   └── ports/                # Interface definitions (VMAdapter)
└── pkg/                      # Public API library
```

### Two-Level Abstraction

1. **Infrastructure Layer** (CURRENT): VM runtimes
   - `native` - HyperKit, Hyper-V, KVM
   - `lima` - Lima with vz driver (macOS)
   - `wsl` - Windows Subsystem for Linux
   - `microvm` - Firecracker microVM
   - `wasm` - WebAssembly runtime

2. **Platform Target Layer** (PLANNED): Target platforms
   - Apple: iOS, tvOS, watchOS, visionOS, macOS
   - Android: Phone, Tablet, Wear, TV, Automotive
   - Smart TV: Tizen, webOS, Roku, Fire TV
   - Gaming: PlayStation, Switch, Xbox
   - AR/VR: Quest, HoloLens, SteamVR, SteamOS

## Quick Start

```bash
# Clone and build
git clone https://github.com/KooshaPari/nanovms.git
cd nanovms
go build ./cmd/nanovms

# Probe system capabilities
./nanovms probe

# Create a sandbox with specified VM flavor
./nanovms create dev --vm-flavor lima --image ubuntu:22.04
./nanovms create secure --vm-flavor microvm --image ubuntu:22.04
```

## Supported Platforms

### Infrastructure Adapters

| Adapter | Platform | Status |
|---------|----------|--------|
| `mac/` | macOS | ✅ Implemented (Lima/VZ) |
| `windows/` | Windows | ⚠️ Partial (WSL2 stub) |
| `linux/` | Linux | ⚠️ Partial (Native/KVM stub) |
| `microvm/` | All | 📋 Planned (Firecracker) |
| `wasm/` | All | ⚠️ Partial (WASM stub) |
| `sandbox/` | Linux | ✅ Implemented (bwrap, firejail) |

### Planned: Platform Targets

| Category | Targets | Infrastructure |
|----------|---------|----------------|
| Apple | iOS, iPadOS, tvOS, watchOS, visionOS | Lima/VZ |
| Android | Phone, Tablet, Wear, TV, Auto | WSL2/Lima |
| Smart TV | Tizen, webOS, Roku, Fire TV | QEMU/Lima |
| Gaming | PlayStation, Switch, Xbox | Various |
| AR/VR | Quest, HoloLens, SteamVR, SteamOS | Remote/QEMU |
| IoT | Raspberry Pi, Pine64, ESP32, FreeRTOS | QEMU |

## VM Tiers

| Tier | Technology | Use Case | Overhead |
|------|------------|----------|----------|
| 1 - Native | HyperKit, Hyper-V, KVM | Production parity | Highest |
| 2 - Lima/WSL | Lima/VZ, WSL2 | Daily development | Medium |
| 3 - MicroVM | Firecracker, Cloud Hypervisor | CI/CD, isolation | Lowest |
| 4 - WASM | Wasmtime, Wasmer | Lightweight execution | Minimal |

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

## License

MIT
