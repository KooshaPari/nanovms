<!-- AI-DD-META:START -->
<!-- This repository is planned, maintained, and managed by AI Agents only. -->
<!-- Slop issues are expected and intentionally present as part of an HITL-less -->
<!-- /minimized AI-DD metaproject of learning, refining, and building brute-force -->
<!-- training for both agents and the human operator. -->
![Downloads](https://img.shields.io/github/downloads/KooshaPari/nanovms/total?style=flat-square&label=downloads&color=blue)
![GitHub release](https://img.shields.io/github/v/release/KooshaPari/nanovms?style=flat-square&label=release)
![License](https://img.shields.io/github/license/KooshaPari/nanovms?style=flat-square)
![AI-Slop](https://img.shields.io/badge/AI--DD-Slop%20Expected-orange?style=flat-square)
![AI-Only-Maintained](https://img.shields.io/badge/Planned%20%26%20Maintained%20by-AI%20Agents%20Only-red?style=flat-square)
![HITL-less](https://img.shields.io/badge/HITL--less%20AI--DD-metaproject-yellow?style=flat-square)

> ⚠️ **AI-Agent-Only Repository**
>
> This repo is **planned, maintained, and managed exclusively by AI Agents**.
> Slop issues, rough edges, and AI artifacts are **expected and intentionally
> present** as part of an **HITL-less / minimized AI-DD** metaproject focused
> on learning, refining, and brute-force training both the agents and the
> human operator. Bug reports and contributions are still welcome, but please
> expect AI-generated code, comments, and documentation throughout.
<!-- AI-DD-META:END -->
# NVMS - NanoVM Service (Unified)

[![AI Slop Inside](https://sladge.net/badge.svg)](https://sladge.net)

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
- [Architecture](docs/reference/architecture.md)
- [Quickstart Guide](docs/guides/quickstart.md)
- [Implementation Roadmap](docs/implementation-roadmap.md)
- [ADR Index](docs/adr/)

## License

Apache-2.0
