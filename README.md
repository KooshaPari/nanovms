# NVMS - NanoVM Service (Unified)

> **Merged Implementation**: KooshaPari/nanovms + BytePort/nvms + PhenoCompose Driver

NVMS provides **3-tier isolation** for secure, efficient application deployment:
- **Tier 1 (WASM)**: ~1ms startup, fast tools, trusted code
- **Tier 2 (gVisor)**: ~90ms startup, browser automation, semi-trusted
- **Tier 3 (Firecracker)**: ~125ms startup, full isolation, untrusted code

## Quick Start

```bash
# Deploy with NVMS
nvms deploy --tier 1 --config nvms.yaml  # WASM
nvms deploy --tier 2 --config nvms.yaml  # gVisor
nvms deploy --tier 3 --config nvms.yaml  # Firecracker

# Or use PhenoCompose (unified interface)
pheno-compose deploy --runtime nvms --config nvms.yaml
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    UNIFIED NVMS STACK                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │ PhenoCompose│    │   NVMS CLI  │    │  BytePort   │    │
│  │   (Rust)    │    │    (Go)     │    │   (Go)      │    │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    │
│         │                  │                  │            │
│         └──────────────────┴──────────────────┘            │
│                            │                                │
│                    ┌───────▼───────┐                        │
│                    │   NVMS Core   │                        │
│                    │    (Merged)   │                        │
│                    └───────┬───────┘                        │
│                            │                                │
│         ┌──────────────────┼──────────────────┐            │
│         ▼                  ▼                  ▼            │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐        │
│  │    WASM    │    │   gVisor   │    │ Firecracker│        │
│  │  (~1ms)    │    │  (~90ms)   │    │  (~125ms)  │        │
│  └────────────┘    └────────────┘    └────────────┘        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Merge History

| Component | Source | Status | Contribution |
|-----------|--------|--------|--------------|
| **Core 3-tier isolation** | KooshaPari/nanovms | ✅ Complete | WASM/gVisor/Firecracker |
| **AWS deployment** | BytePort/nvms | ✅ Merged | Firecracker orchestration |
| **Unified interface** | PhenoCompose | ✅ New | Rust driver, standardization |

## Platform Support

| Platform | Tier 1 (WASM) | Tier 2 (gVisor) | Tier 3 (Firecracker) |
|----------|---------------|-----------------|----------------------|
| **macOS** | ✅ Native | ✅ Lima/VZ | ✅ Virtualization.framework |
| **Linux** | ✅ Native | ✅ Native | ✅ KVM |
| **Windows** | ✅ Native | ✅ WSL2 | ✅ WSL2 |

## Installation

```bash
# Install NVMS
curl -fsSL https://get.nvms.dev | sh

# Or build from source
git clone https://github.com/KooshaPari/nvms.git
cd nvms && go build ./cmd/nvms

# Install PhenoCompose driver
cargo install pheno-compose --features nvms-driver
```

## Documentation

- [PhenoCompose Integration](integrations/pheno-compose/README.md)
- [AWS Deployment](docs/aws-deployment.md)
- [Architecture](docs/architecture.md)

## License

Apache-2.0
