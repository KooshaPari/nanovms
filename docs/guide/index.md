# NanoVMS - Nano Virtual Machine Services

Lightweight, headless VM abstraction for agents — supports desktop, mobile simulators, and emerging form factors.

## Features

- **Multi-Platform Support**: macOS, Windows, Linux + mobile simulators (iOS, Android, tvOS, watchOS, VisionOS)
- **Headless IDE Support**: Run Android Studio / Xcode in VMs for agent use
- **Multi-Tier VM Architecture**: Native VMs → Container/WSL → MicroVMs (Firecracker)
- **Sandbox Isolation**: gVisor, landlock, seccomp, WASM runtime layers
- **Simulator Abstraction**: Unified interface for iOS Simulator, Android Emulator, tvOS, watchOS, VisionOS

## Quick Start

```bash
# Clone the repository
git clone https://github.com/KooshaPari/nanovms.git
cd nanovms

# Build
go build ./cmd/nanovms

# Run
./nanovms --help
```

## Architecture

NanoVMS uses a hexagonal (ports and adapters) architecture:

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
| iOS Simulator | Via macOS host | Lima VM | Stable |
| Android Emulator | Headless mode | Via Lima | Stable |

## License

MIT
