# NanoVMS Specification (2026)

> Nano Virtual Machine Services — SOTA Hypervisor Abstraction for Agents & Game Automation

## Executive Summary

NanoVMS is a **full virtualization replacement** for all project/agent needs, targeting:
- **Project Desktop Environments** (headless, remote display)
- **Agent Computer Use** (sandboxed execution)
- **Game Automation Testing** (parallel, <10s startup)
- **Cross-platform development** (iOS, Android, Windows, Linux, macOS)
- **GPU-intensive workloads** (VFIO passthrough, Looking Glass)

Personal gaming VMs use UTM/KVM directly (bypassing NanoVMS).

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              NanoVMS Stack                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        User Space                                      │   │
│  │                                                                      │   │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐  │   │
│  │   │   NanoVMS   │  │   NanoVMS   │  │   NanoVMS   │  │  NanoVMS  │  │   │
│  │   │    CLI      │  │   Agent     │  │    API      │  │  Web UI   │  │   │
│  │   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬─────┘  │   │
│  │          │                │                │                │        │   │
│  │          └────────────────┴────────────────┴────────────────┘        │   │
│  │                                   │                                     │   │
│  │                          ┌────────▼────────┐                         │   │
│  │                          │   Orchestrator   │                         │   │
│  │                          │   (Go/Rust FFI)  │                         │   │
│  │                          └────────┬────────┘                         │   │
│  │                                   │                                   │   │
│  └───────────────────────────────────┼───────────────────────────────────┘   │
│                                      │                                       │
│  ┌───────────────────────────────────┼───────────────────────────────────┐   │
│  │                          Hypervisor Layer                              │   │
│  │                                                                      │   │
│  │  ┌────────────────────────────────────────────────────────────────┐  │   │
│  │  │                    VM Flavor Adapters                            │  │   │
│  │  │                                                                 │  │   │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │  │   │
│  │  │  │ VFIO/    │  │ MicroVM  │  │ Container│  │   WASM      │  │  │   │
│  │  │  │ GPU Passthrough│ │ (FC/CH)  │  │ (runc)  │  │  Runtime    │  │  │   │
│  │  │  └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │  │   │
│  │  │                                                                 │  │   │
│  │  │  ┌────────────────────────────────────────────────────────┐   │  │   │
│  │  │  │              Sandbox Isolation Layer                      │   │  │   │
│  │  │  │   bwrap │ gVisor │ landlock │ seccomp │ AppArmor     │   │  │   │
│  │  │  └────────────────────────────────────────────────────────┘   │  │   │
│  │  │                                                                 │  │   │
│  │  └────────────────────────────────────────────────────────────────┘  │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         Kernel / Hardware                              │   │
│  │                                                                      │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────────────┐  │   │
│  │  │  KVM/QEMU      │  │  VFIO-PCI      │  │   Looking Glass     │  │   │
│  │  │  (Linux)       │  │  (GPU Pass)     │  │   (Display)        │  │   │
│  │  └────────────────┘  └────────────────┘  └──────────────────────┘  │   │
│  │                                                                      │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────────────┐  │   │
│  │  │  HyperKit      │  │  Apple HV      │  │   Hyper-V/WHPX     │  │   │
│  │  │  (macOS VM)   │  │  (Apple Silicon)│  │   (Windows)        │  │   │
│  │  └────────────────┘  └────────────────┘  └──────────────────────┘  │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## VM Flavor Taxonomy (2026)

### Tier 1: Native VM (Full Virtualization)

**Use Cases:**
- GPU-intensive workloads (game automation, ML training)
- Windows game testing with near-bare-metal performance
- Any workload requiring VFIO GPU passthrough
- Steam game automation with <5% overhead

**Implementations:**

| Platform | Hypervisor | VFIO Support | SR-IOV | Looking Glass |
|----------|------------|--------------|---------|---------------|
| **Linux** | KVM/QEMU | ✅ Native | ✅ Native | ✅ IVSHMEM |
| **macOS** | HyperKit | ❌ | ❌ | N/A |
| **Windows** | Hyper-V + WHPX | ⚠️ Discrete GPU only | ❌ | ⚠️ RemoteFX |

**VFIO Configuration:**
```yaml
# .nanovms/vfio-config.yaml
vfio:
  enabled: true
  primary_gpu: "10de:2684"  # NVIDIA RTX 4090
  iommu_group: 1
  looking_glass:
    enabled: true
    ivshmem_size: 256MB
    window_title: "NanoVMS VM Display"
  performance:
    cpu_pinning: true
    huge_pages: 2M
    memlock: unlimited
    cpu_governor: performance
```

**Performance Targets:**
| Metric | Target | VFIO Bare Metal |
|--------|--------|------------------|
| GPU FPS | ≥95% | 100% baseline |
| CPU Performance | ≥99% | 100% baseline |
| Memory Latency | ≤5% overhead | baseline |
| Startup Time | 10-30s | N/A |

---

### Tier 2: MicroVM (Minimal Overhead)

**Use Cases:**
- Agent computer use (sandboxed execution)
- Parallel CI/CD runners
- Serverless-style workloads
- Ephemeral environments

**Implementations:**

| Runtime | Language | Startup | Memory | Security |
|---------|----------|---------|--------|----------|
| **Firecracker** | Rust | <125ms | <5MB | KVM+jailer |
| **Cloud Hypervisor** | Rust | <150ms | <10MB | KVM |
| **QEMU (micro mode)** | C | <500ms | <32MB | KVM |
| **Kata Containers** | Go/Rust | <1s | <50MB | KVM+guest |

**Configuration:**
```yaml
# .nanovms/microvm-config.yaml
microvm:
  runtime: firecracker
  vcpus: 2
  memory_mb: 1024
  kernel:
    path: /var/lib/nanovms/vmlinux
    boot_args: "console=ttyS0 reboot=k panic=1"
  rootfs:
    path: /var/lib/nanovms/rootfs.ext4
    size_gb: 20
  snapshot:
    enabled: true
    base_image: /var/lib/nanovms/base.img
```

---

### Tier 3: Container Runtime

**Use Cases:**
- Local development
- Fast iteration (seconds vs minutes)
- Linux-only workloads
- Resource-constrained environments

**Implementations:**

| Runtime | Isolation | Startup | Memory | OCI Compatible |
|---------|----------|---------|--------|----------------|
| **runc** | namespaces | ~500ms | ~0MB | ✅ |
| **runc + gVisor** | ptrace | ~100ms | ~50MB | ✅ |
| **bubblewrap** | namespaces | <10ms | ~0MB | ❌ |
| **firejail** | AppArmor | <50ms | ~1MB | ❌ |

---

### Tier 4: WASM Runtime

**Use Cases:**
- Language-agnostic execution
- Plugin systems
- Edge computing
- Zero-install execution

**Implementations:**

| Runtime | Language | Startup | Memory | WASI Support |
|---------|----------|---------|--------|--------------|
| **Wasmtime** | Rust | <5ms | <1MB | ✅ |
| **Wasmer** | C/Rust | <3ms | <1MB | ✅ |
| **WAMR** | C | <1ms | <500KB | ✅ |
| **WasmEdge** | C++ | <5ms | <1MB | ✅ |

---

## Game Automation Testing Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Game Automation Test Cluster                              │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    NanoVMS Orchestrator                                │   │
│  │                                                                      │   │
│  │   ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐   │   │
│  │   │  Test Scheduler  │  │  Resource Pool  │  │  Result Agg.   │   │   │
│  │   │  (Temporal)      │  │  (VFIO/KVM)    │  │  (Postgres)   │   │   │
│  │   └────────┬─────────┘  └────────┬─────────┘  └────────┬────────┘   │   │
│  │            │                     │                     │            │   │
│  │            └─────────────────────┼─────────────────────┘            │   │
│  │                                  │                                  │   │
│  └──────────────────────────────────┼──────────────────────────────────┘   │
│                                     │                                       │
│  ┌──────────────────────────────────┼──────────────────────────────────┐   │
│  │                         VM Node Pool                                  │   │
│  │                                                                      │   │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │   │
│  │   │  Game VM 1  │  │  Game VM 2  │  │  Game VM 3  │  │  ...    │ │   │
│  │   │  (Windows)  │  │  (Windows)  │  │  (Windows)  │  │         │ │   │
│  │   │  VFIO GPU   │  │  VFIO GPU   │  │  VFIO GPU   │  │         │ │   │
│  │   │  Looking Glass│ │  Looking Glass│ │  Looking Glass│ │         │ │   │
│  │   └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Fast Provisioning Layer                             │   │
│  │                                                                      │   │
│  │   ┌────────────────┐  ┌────────────────┐  ┌──────────────────────┐  │   │
│  │   │  Pre-copied    │  │  Compressed    │  │   RAM Cloud          │  │   │
│  │   │  Base Images   │  │  Snapshots     │  │   (tmpfs)           │  │   │
│  │   │  (NFS/cached) │  │  (zstd)       │  │                     │  │   │
│  │   └────────────────┘  └────────────────┘  └──────────────────────┘  │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Fast VM Startup Strategy

**Target: <10 seconds from cold start to game running**

| Phase | Technique | Time Target |
|-------|-----------|------------|
| 1. VM fork | Copy-on-write snapshot | <1s |
| 2. Memory init | Pre-allocated RAM disk | <1s |
| 3. GPU attach | VFIO rebind | <2s |
| 4. Kernel boot | pvops/UEFI fast boot | <3s |
| 5. Steam launch | Pre-warmed process | <3s |
| **Total** | | **<10s** |

**Implementation:**
```bash
# Pre-warm sequence
1. Create base VM image with Windows + Steam installed
2. Snapshot compressed with zstd (50GB → ~15GB)
3. On request:
   a. Copy COW diff (~100MB)
   b. Extract to RAM disk (~5s for 20GB)
   c. Start VM with Looking Glass (~3s)
   d. Launch Steam headless (~2s)
```

---

## Platform Support Matrix

### Host Platforms

| Platform | VFIO | MicroVM | Container | WASM | GPU Passthrough |
|----------|------|---------|-----------|------|-----------------|
| **Linux (KVM)** | ✅ Full | ✅ Firecracker | ✅ runc | ✅ Wasmtime | ✅ Native |
| **macOS (Apple Silicon)** | N/A | ✅ FC | ⚠️ Rosetta | ✅ Wasmtime | N/A |
| **macOS (Intel)** | N/A | ✅ FC | ⚠️ Lima | ✅ Wasmtime | N/A |
| **Windows (Hyper-V)** | ⚠️ Discrete | ✅ FC | ⚠️ WSL2 | ✅ Wasmtime | ⚠️ RemoteFX |

### Guest Platforms

| Guest | VFIO (KVM) | MicroVM | Container | WASM | Looking Glass |
|-------|-------------|---------|-----------|------|--------------|
| **Windows 10/11** | ✅ | ✅ | ❌ | ❌ | ✅ |
| **Linux** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **macOS** | ⚠️ (hackintosh) | ❌ | ❌ | ❌ | ⚠️ |
| **SteamOS** | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Android** | ✅ | ✅ | ⚠️ | ❌ | ⚠️ |
| **WASI (Linux)** | ❌ | ❌ | ✅ | ✅ | ❌ |

---

## Security Model

### Isolation Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Security Boundaries                                   │
│                                                                             │
│   Least Trusted ─────────────────────────────────────── Most Trusted         │
│                                                                             │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│   │  WASM    │  │ Process  │  │Container │  │ MicroVM  │  │  VFIO    │   │
│   │  Runtime │  │ (gVisor) │  │ (runc)   │  │ (FC)     │  │  (KVM)   │   │
│   │          │  │          │  │          │  │          │  │          │   │
│   │ Bytecode │  │ Syscall  │  │namespace │  │ KVM+     │  │ KVM+     │   │
│   │ sandbox  │  │ filter   │  │ cgroup   │  │ jailer   │  │ IOMMU    │   │
│   │          │  │          │  │          │  │          │  │          │   │
│   │ ✓✓✓✓✓✓✓ │  │ ✓✓✓✓✓✓✓ │  │ ✓✓✓✓✓✓✓ │  │ ✓✓✓✓✓✓✓ │  │ ✓✓✓✓✓✓✓ │   │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│                                                                             │
│   Can be stacked: WASM → gVisor → MicroVM → VFIO                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### VFIO Security Requirements

```yaml
# Security requirements for GPU passthrough
vfio_security:
  iommu: enabled          # Required: Intel VT-d or AMD-Vi
  efi_firmware: signed   # Required: UEFI Secure Boot
  smm: enabled           # Required: System Management Mode isolation
  dma_protection: strict  # Required: All DMA requests go through IOMMU
  rollback_protection: enabled  # Optional: TPM PCR integrity
```

---

## Performance Benchmarks (2026)

### VM Startup Comparison

| VM Type | Cold Start | Warm Start | Memory | Use Case |
|---------|------------|------------|--------|----------|
| **VFIO (Windows)** | 10-30s | 3-5s | 4-32GB | Game testing |
| **MicroVM (FC)** | <1s | <100ms | 128-512MB | Agent workloads |
| **Container (runc)** | ~500ms | ~100ms | ~0MB | Local dev |
| **Container + gVisor** | ~800ms | ~200ms | ~50MB | Untrusted code |
| **WASM (Wasmtime)** | <5ms | <1ms | <1MB | Plugins |

### GPU Passthrough Performance

| Metric | Bare Metal | VFIO VM | Delta |
|--------|------------|---------|-------|
| 3DMark Score | 25000 | 24500 | -2% |
| Gaming FPS (avg) | 144 | 142 | -1.4% |
| Gaming FPS (1%) | 120 | 118 | -1.7% |
| Memory Latency | 60ns | 62ns | +3.3% |
| GPU Compute | 100% | 99% | -1% |

### Game Automation Test Throughput

| Configuration | VMs/Host | Start Time | Throughput |
|--------------|----------|------------|------------|
| **VFIO + Looking Glass** | 1-2 | 15-30s | 2-4 tests/hour |
| **MicroVM + Wine** | 8-16 | 2-5s | 50-100 tests/hour |
| **Container + Xvfb** | 32-64 | 500ms | 200-400 tests/hour |
| **WASM (Linux games)** | 100+ | <10ms | 1000+ tests/hour |

---

## Service Integrations

### Database Options

| Database | Use Case | Complexity |
|----------|----------|------------|
| **SQLite** | Local dev, single node | Low |
| **PostgreSQL** | Team server, analytics | Medium |
| **TimescaleDB** | VM metrics, long-term retention | Medium |
| **S3-compatible** | VM image storage | Low |

### Message Bus

| Service | Use Case | Complexity |
|---------|----------|------------|
| **NATS** | Real-time events, streaming | Low |
| **Kafka** | High-throughput event log | High |
| **Redis** | Caching, pub/sub | Low |

### Workflow Orchestration

| Service | Use Case | Complexity |
|---------|----------|------------|
| **Temporal** | Multi-step VM provisioning | High |
| **Hatchet** | Self-hosted workflows | Medium |
| **Dagster** | Data pipeline + VMs | Medium |

---

## CLI Commands

### Core Commands

```bash
# VM lifecycle
nanovms create <name> --flavor vfio --gpu auto
nanovms start <name>
nanovms stop <name>
nanovms delete <name>
nanovms list

# Game automation
nanovms game launch <name> --steam-tokens /path/to/tokens
nanovms game test <name> --test-suite automated-tests
nanovms game record <name> --output ./recordings

# Display (Looking Glass)
nanovms display attach <name>
nanovms display list

# Image management
nanovms image pull ubuntu-24.04
nanovms image snapshot <name> --name my-base
nanovms image compress <name> --algo zstd

# Agent commands
nanovms agent spawn --flavor microvm --sandbox gvisor
nanovms agent exec <id> -- npm test
nanovms agent destroy <id>
```

---

## File Structure

```
nanovms/
├── cmd/
│   └── nanovms/              # CLI entry point
├── internal/
│   ├── adapters/             # VM flavor adapters
│   │   ├── vfio/            # KVM/VFIO adapter
│   │   ├── firecracker/      # Firecracker adapter
│   │   ├── container/        # runc adapter
│   │   ├── wasm/            # WASM adapter
│   │   └── sandbox/          # gVisor, bwrap, etc.
│   ├── domain/               # Core types
│   └── ports/               # Interfaces
├── pkg/
│   ├── api/                  # Public API
│   ├── orchestrator/         # VM orchestration
│   └── image/                # Image management
├── .nanovms/
│   ├── config.yaml           # Global config
│   ├── vfio-config.yaml      # VFIO settings
│   └── profiles/             # VM profiles
├── SPEC.md
└── README.md
```

---

## Quality Gates

```bash
# Format and lint
go fmt ./...
go vet ./...
golangci-lint run

# Build
go build ./...

# Test
go test ./... -v -race

# Benchmark
go test ./... -bench=. -benchmem

# VFIO sanity check
nanovms hwtest --vfio
```

---

## Status Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Implemented and tested |
| ⚠️ | Partial implementation (stub) |
| 📋 | Planned (not started) |
| ❌ | Not supported / N/A |

---

## 2026 Research Sources

- Firecracker v1.8: https://github.com/firecracker-microvm/firecracker
- Cloud Hypervisor v37: https://github.com/cloud-hypervisor/cloud-hypervisor
- Looking Glass vB5: https://looking-glass.io
- QEMU v8.2: https://www.qemu.org
- Linux KVM: https://www.linux-kvm.org
- VFIO: https://docs.kernel.org/vfio.html
- Kata Containers v3: https://katacontainers.io
- gVisor: https://gvisor.dev
- Incus v6: https://linuxcontainers.org/incus
- Wasmtime v23: https://wasmtime.dev

---

*Last updated: 2026-04-02*
