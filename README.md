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

## Tiers (30)

After the 15 -> 30 tier expansion, NVMS exposes 30 sandbox adapters in
`pkg/tier`. Each adapter implements the same `tier.Adapter` interface
(`Probe`, `Deploy`, `Start`, `Stop`, `Delete`, `GetStartupTime`,
`Info`) and is registered in `tier.DefaultRegistry()`.

The original 15 tiers are unchanged:

| # | Name | Security | Startup | Notes |
|---|------|----------|---------|-------|
| 1  | wasm         | low      | 1ms     | WASI / Wasmtime |
| 2  | gvisor       | medium   | 90ms    | runsc user-space kernel |
| 3  | firecracker  | high     | 125ms   | microVM via KVM |
| 4  | landlock     | low      | 1ms     | Linux Landlock LSM |
| 5  | seccomp      | low      | 1ms     | seccomp-bpf |
| 6  | native       | low      | 0ms     | in-process |
| 7  | docker       | medium   | 600ms   | Docker Engine |
| 8  | podman       | medium   | 700ms   | Podman (rootless-friendly) |
| 9  | hyperkit     | high     | 400ms   | macOS hyperkit (deprecated by applevz) |
| 10 | applevz      | high     | 250ms   | Virtualization.framework |
| 11 | lima         | high     | 3000ms  | Lima Linux VM on macOS |
| 12 | kvm          | high     | 150ms   | Linux KVM |
| 13 | qemu         | high     | 2000ms  | QEMU full emulation |
| 14 | cloudhv      | high     | 180ms   | Microsoft Cloud Hypervisor |
| 15 | crosvm       | high     | 200ms   | Chromium OS Virtual Machine Monitor |

The 15 new tiers introduced in the expansion:

| # | Name | Security | Startup | Host | Notes |
|---|------|----------|---------|------|-------|
| 16 | kata           | high     | 250ms   | Linux   | Kata Containers (containerd-shim-kata-v2) |
| 17 | youki          | medium   | 30ms    | Linux   | youki Rust OCI runtime |
| 18 | systemdnspawn  | medium   | 100ms   | Linux   | systemd-nspawn container manager |
| 19 | nitrorekf      | untrusted| 5000ms  | Linux   | AWS Nitro Enclaves (`/dev/nitro_enclaves`) |
| 20 | sev            | untrusted| 500ms   | Linux   | AMD SEV encrypted VMs (`/dev/sev`) |
| 21 | tdx            | untrusted| 600ms   | Linux   | Intel TDX trusted domains (`/dev/tdx-guest`) |
| 22 | kubevirt       | high     | 2000ms  | Linux   | KubeVirt (k8s-managed VMs) |
| 23 | virtcontainers | medium   | 200ms   | Linux   | Intel Clear Containers (legacy virtcontainers) |
| 24 | jail           | medium   | 50ms    | FreeBSD | FreeBSD jail(8) |
| 25 | pledge         | low      | 1ms     | OpenBSD | OpenBSD pledge(2) |
| 26 | sandboxexec    | low      | 5ms     | macOS   | macOS `sandbox-exec` with SBPL profile |
| 27 | gvisordocker   | medium   | 600ms   | Linux, macOS | gVisor (`runsc`) wrapped in Docker |
| 28 | wolfi          | medium   | 50ms    | Linux   | Wolfi Linux (apk-based distroless) |
| 29 | distroless     | high     | 80ms    | Linux, macOS, Windows | Google's distroless container images |
| 30 | userns         | low      | 1ms     | Linux   | Linux user namespaces (unprivileged) |

### Profiles

`tier.ProfilePolicy` maps `NVMS_PROFILE` to a (security, platform,
startup-budget) bundle:

- `dev`         - Security=Low, Platform=Linux, 5s budget (wasm/native).
- `ci`          - Security=Medium, Platform=Linux, 10s budget.
- `ci-secure`   - Security=Medium, Platform=Linux, 10s budget; prefers landlock/seccomp/userns/gvisor.
- `ci-fast`     - Security=Low, Platform=Linux, 2s budget; prefers native/seccomp/userns.
- `prod-secure` - Security=Untrusted, no budget; demands a hardware VM (firecracker/qemu/cloudhv/sev/tdx/nitro).
- `prod-fast`   - Security=High, 1s budget; cheapest high-security Linux tier.
- `airgapped`   - Security=High, no budget; no live cloud probes.

### CLI

```bash
nvms tier list                    # table of all registered tiers
nvms tier list --json             # JSON of all registered tiers
nvms tier list --security high    # filter by security level
nvms tier info firecracker        # show one tier's metadata
nvms tier probe kata              # probe a single tier's runtime
```

Note: on a single host GOOS, the count of registered tiers is at most
28 because three new tiers are OS-gated (`jail` -> freebsd only,
`pledge` -> openbsd only, `sandboxexec` -> darwin only). The
canonical metadata in `tier.allTiers()` always lists all 30.

## License

Apache-2.0
