# NanoVMS Implementation Plan

## Overview

NanoVMS is a cloud infrastructure virtualization platform targeting SOTA performance for development, testing, and production workloads on consumer hardware.

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         NanoVMS Hypervisor Stack                            │
│                                                                             │
│  Tier 0: Bare Metal (VFIO) ──► GPU Passthrough, Near-Bare-Metal Gaming     │
│  Tier 1: MicroVM (Firecracker) ──► <125ms startup, 150 VMs/second          │
│  Tier 2: Heavy Containers (Kata/gVisor) ──► Secure syscall isolation       │
│  Tier 3: Lightweight Sandboxes (bwrap/firejail) ──► <10ms startup         │
│  Tier 4: Process Isolation (unshare) ──► <2ms startup                    │
│  Tier 5: WASM Runtime ──► Language-level sandboxing                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Core Infrastructure (Q1 2026)

#### 1.1 Project Setup
- [x] Project structure
- [x] Go module initialization
- [x] CI/CD configuration
- [x] VitePress documentation setup
- [x] SPEC.md with comprehensive research

#### 1.2 Core Domain Models
- [ ] VM types and states
- [ ] Sandbox types and configurations
- [ ] Isolation tier definitions
- [ ] Resource configuration schemas

#### 1.3 Adapter Interfaces
- [ ] VMAdapter interface
- [ ] SandboxAdapter interface
- [ ] StorageAdapter interface
- [ ] NetworkAdapter interface

### Phase 2: VM Adapters (Q1-Q2 2026)

#### 2.1 Firecracker Adapter
- [ ] Firecracker binary detection
- [ ] VM lifecycle management (create, start, stop, delete)
- [ ] Snapshot support
- [ ] Memory ballooning
- [ ] vCPU management
- [ ] Network configuration

#### 2.2 Linux Adapter
- [ ] Native namespace support (unshare)
- [ ] cgroups v2 integration
- [ ] Rootless container support
- [ ] Overlay filesystem
- [ ] Volume management

#### 2.3 macOS Adapter
- [ ] Lima/Colima integration
- [ ] Virtualization.framework
- [ ] Rosetta 2 support (ARM)

#### 2.4 Windows Adapter
- [ ] WSL2 integration
- [ ] Hyper-V support
- [ ] Cloud Hypervisor support

### Phase 3: Sandbox Adapters (Q2 2026)

#### 3.1 gVisor Adapter
- [ ] runsc binary detection
- [ ] Sentry process management
- [ ] Gofer filesystem proxy
- [ ] Network proxy

#### 3.2 landlock Adapter
- [ ] Landlock LSM detection
- [ ] Filesystem restriction rules
- [ ] Rule merging

#### 3.3 bwrap/firejail Adapter
- [ ] bubblewrap integration
- [ ] firejail profile execution
- [ ] Namespace isolation

#### 3.4 seccomp Adapter
- [ ] seccomp-bpf profile generation
- [ ] Syscall allowlisting
- [ ] Error handling

### Phase 4: Game Automation (Q2-Q3 2026)

#### 4.1 Game VM Templates
- [ ] Steam headless installation
- [ ] Game automation framework
- [ ] Headless browser support

#### 4.2 Snapshot System
- [ ] Pre-copied compressed images
- [ ] Instant resume from snapshots
- [ ] Differential snapshots

#### 4.3 Parallel Test Execution
- [ ] Test runner orchestration
- [ ] Resource allocation
- [ ] Result aggregation

### Phase 5: Consumer Hardware Optimization (Q3 2026)

#### 5.1 Hardware Profiles
- [ ] Budget profile (4C/8T, 16GB RAM)
- [ ] Mid profile (8C/16T, 32GB RAM)
- [ ] Enthusiast profile (16C/32T, 64GB RAM)

#### 5.2 Performance Tuning
- [ ] CPU P-state optimization
- [ ] Huge page allocation
- [ ] NUMA binding
- [ ] IRQ balancing

### Phase 6: Production Hardening (Q4 2026)

#### 6.1 Observability
- [ ] Prometheus metrics
- [ ] OpenTelemetry tracing
- [ ] Structured logging

#### 6.2 High Availability
- [ ] Health checks
- [ ] Automatic failover
- [ ] State persistence

#### 6.3 Security
- [ ] Confidential computing support (TDX/SGX)
- [ ] Signed VM images
- [ ] Audit logging

## File Structure

```
nanovms/
├── cmd/
│   └── nanovms/
│       └── main.go
├── internal/
│   ├── adapters/
│   │   ├── firecracker/
│   │   │   └── adapter.go
│   │   ├── linux/
│   │   │   └── adapter.go
│   │   ├── mac/
│   │   │   └── adapter.go
│   │   ├── windows/
│   │   │   └── adapter.go
│   │   ├── sandbox/
│   │   │   ├── gvisor.go
│   │   │   ├── landlock.go
│   │   │   ├── bwrap.go
│   │   │   └── seccomp.go
│   │   └── wasm/
│   │       └── adapter.go
│   ├── core/
│   │   ├── vm.go
│   │   ├── sandbox.go
│   │   ├── storage.go
│   │   ├── network.go
│   │   └── scheduler.go
│   ├── domain/
│   │   ├── vm.go
│   │   ├── sandbox.go
│   │   ├── config.go
│   │   └── types.go
│   └── ports/
│       ├── vm_adapter.go
│       ├── sandbox_adapter.go
│       ├── storage_adapter.go
│       └── network_adapter.go
├── pkg/
│   ├── vfio/
│   │   └── manager.go
│   ├── snapshot/
│   │   └── manager.go
│   └── metrics/
│       └── collector.go
├── docs/
│   ├── guide/
│   ├── api/
│   ├── adr/
│   └── research/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── benchmarks/
└── scripts/
    ├── build.sh
    └── test.sh
```

## CLI Commands

### VM Management
```bash
# Create VM
nanovms vm create --name=dev --flavor=firecracker --cpu=4 --memory=8GB

# Start VM
nanovms vm start dev

# Stop VM
nanovms vm stop dev

# Delete VM
nanovms vm delete dev

# List VMs
nanovms vm list

# SSH into VM
nanovms vm ssh dev

# Snapshot VM
nanovms vm snapshot dev --name=base

# Restore snapshot
nanovms vm restore dev --snapshot=base
```

### Sandbox Management
```bash
# Create sandbox
nanovms sandbox create --name=test --tier=gvisor

# Apply sandbox to VM
nanovms sandbox apply --vm=dev --sandbox=test

# List sandboxes
nanovms sandbox list
```

### Game Automation
```bash
# Create game VM
nanovms game create --name=test-game --steam --headless

# Run parallel tests
nanovms game test --suite=regression --parallel=8

# Snapshot game state
nanovms game snapshot test-game --name=level-1-complete
```

### VFIO/GPU Passthrough
```bash
# List available GPUs
nanovms vfio list-gpus

# Bind GPU to VFIO
nanovms vfio bind --gpu=01:00.0 --driver=nvidia

# Create gaming VM with GPU
nanovms vm create --name=gaming --flavor=firecracker --gpu-passthrough

# Connect via Looking Glass
nanovms lookingglass connect gaming
```

## Performance Targets

| Metric | Target | Method |
|--------|--------|--------|
| MicroVM startup | <125ms | Firecracker snapshots |
| Container startup | <500ms | Pre-warmed |
| Sandbox startup | <10ms | bwrap/firejail |
| Process isolation | <2ms | unshare |
| WASM cold start | <1ms | Wasmtime |
| Memory overhead | <5MB/VM | Jailer |
| CPU overhead | <1% | KVM paravirtualization |
| IOPS | 1M+ | io_uring |
| P99 latency | <100μs | DPDK |

## Dependencies

### Required Binaries
- firecracker (or cloud-hypervisor)
- runc or youki
- bwrap
- firejail (optional)
- iproute2
- dnsmasq
- qemu-img

### Go Dependencies
- github.com/firecracker-microvm/firecracker-go-sdk
- github.com/rootless-containers/rootlesskit
- github.com/containers/bubblewrap
- github.com/google/gvisor
- github.com/bytecodealliance/wasmtime
- github.com/prometheus/client_golang
- github.com/spf13/cobra
- github.com/spf13/viper

## Testing Strategy

### Unit Tests
- Adapter implementations
- Domain logic
- Configuration parsing
- VM state machines

### Integration Tests
- VM lifecycle
- Sandbox application
- Network configuration
- Storage operations

### Benchmarks
- VM startup time
- Sandbox overhead
- IOPS performance
- Memory efficiency

## Milestones

| Milestone | Date | Deliverables |
|-----------|------|-------------|
| M1: Core | 2026-02-01 | Project setup, domain models, interfaces |
| M2: Firecracker | 2026-03-01 | Working MicroVM support |
| M3: Sandboxes | 2026-04-01 | gVisor, landlock, bwrap |
| M4: Game VMs | 2026-05-01 | Game automation framework |
| M5: HW Profiles | 2026-06-01 | Consumer hardware optimization |
| M6: Production | 2026-07-01 | Observability, HA, security |

## References

- [SPEC.md](./SPEC.md) - Full specification with SOTA research
- [docs/](./docs/) - VitePress documentation
- [docs/adr/](./docs/adr/) - Architecture Decision Records
- [docs/research/](./docs/research/) - Research documentation
- [docs/specs/](./docs/specs/) - Detailed specifications

## Status

- [x] Phase 1.1: Project Setup
- [ ] Phase 1.2: Core Domain Models
- [ ] Phase 1.3: Adapter Interfaces
- [ ] Phase 2.1-2.4: VM Adapters
- [ ] Phase 3.1-3.4: Sandbox Adapters
- [ ] Phase 4.1-4.3: Game Automation
- [ ] Phase 5.1-5.2: Hardware Optimization
- [ ] Phase 6.1-6.3: Production Hardening
