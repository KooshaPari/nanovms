# AGENTS.md — NanoVMS

## Project Overview

- **Name**: NanoVMS (Nano Virtual Machine Services)
- **Description**: Nano Virtual Machine Services — headless VM abstraction for agents with support for Apple, Android, Smart TV, Gaming, IoT/Embedded, and AR/VR platforms
- **Location**: `/Users/kooshapari/CodeProjects/Phenotype/repos/nanovms`
- **Language Stack**: Go 1.23+
- **Published**: Private (Phenotype org)

## Quick Start

```bash
# Clone and setup
git clone https://github.com/KooshaPari/nanovms.git
cd nanovms
go mod download

# Build
go build ./...

# Run tests
go test ./...

# Run CLI
go run ./cmd/nanovms

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
| **Apple** | | | |
| macOS | Lima/Colima + vz | ✅ Active | Primary dev environment |
| iOS | Xcode Simulator | ✅ Via Lima | iPhone, iPad |
| iPadOS | Xcode Simulator | ✅ Via Lima | iPad development |
| tvOS | Xcode Simulator | ✅ Via Lima | Apple TV apps |
| watchOS | Xcode Simulator | ✅ Via Lima | Apple Watch apps |
| visionOS | Xcode Simulator | ✅ Via Lima | Vision Pro |
| **Android** | | | |
| Phone | Emulator headless | ✅ | Pixel, Samsung, etc. |
| Tablet | Emulator | ✅ | Various form factors |
| Wear OS | Emulator | ✅ | Smartwatch |
| Android TV | TV Emulator | ✅ | Leanback launcher |
| Automotive | Auto Emulator | ✅ | Google Automotive |
| **Smart TV** | | | |
| tvOS | Xcode | ✅ Via Lima | Apple TV |
| Android TV | Android Emulator | ✅ | |
| Samsung Tizen | Tizen Studio | 📋 Planned | Via Lima |
| LG webOS | webOS SDK | 📋 Planned | Via Lima |
| Roku | Roku SDK | 📋 Planned | Via Lima |
| Fire TV | Fire OS Emulator | ✅ | Android-based |
| **Gaming** | | | |
| Nintendo Switch | Yuzu/Ryujinx | 📋 Planned | Via Lima + wine |
| Xbox | Dev Mode | 📋 Planned | Windows UWP |
| PlayStation | DevNet | 📋 Remote | Sony DevNet access |
| **IoT/Embedded** | | | |
| Raspberry Pi | QEMU | 📋 Planned | ARM emulation |
| Pine64 | QEMU | 📋 Planned | ARM64 |
| ESP32/FreeRTOS | QEMU | 📋 Planned | Embedded |
| **AR/VR** | | | |
| visionOS | Xcode | ✅ | Vision Pro |
| SteamVR | Steam | 📋 Planned | Windows VR |
| SteamOS | ChimeraOS | 📋 Planned | Steam Deck |
| Meta Quest | Horizon | 📋 Remote | Stream to headset |
| HoloLens | Emulator | 📋 Planned | Windows Hyper-V |
| Magic Leap | Lab | 📋 Remote | Cloud simulator |
| **Other** | | | |
| Linux | Native + gVisor | ✅ Active | Native Linux |
| Windows | WSL2 + gVisor | ✅ Active | Windows path |

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
nanovms info

# Create sandbox
nanovms create --name myapp --platform mac

# Create mobile simulator
nanovms create --name ios-test --platform mobile --simulator-type ios

# List sandboxes
nanovms list

# Delete sandbox
nanovms delete <sandbox-id>

# Pull OCI image
nanovms pull oci://localhost:5000/myapp:latest
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
