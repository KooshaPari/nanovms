# nanovms
# NanoVMS

NanoVMS is a multi-tier sandbox and virtual machine orchestration engine written in Go. It provides a unified API for creating, managing, and deploying workloads across 30 isolation tiers — from lightweight process isolation (WASM, native) through container runtimes (gVisor, Podman, LXC) to full virtualization (Firecracker, QEMU/KVM).

## Features

- **30-tier isolation registry** — select the right isolation level for your workload
- **Real adapter implementations** — Docker, gVisor, Firecracker, Podman, LXC, and more
- **Policy engine** — automated tier selection based on resource requirements, GPU needs, and security constraints
- **Platform support** — Linux, macOS, Windows with platform-specific adapters
- **Security hardening** — Landlock LSM integration, shell injection prevention, environment isolation
- **CLI** — `nvms run`, `nvms list`, `nvms stop`, `nvms probe` subcommands

## Quick Start

```bash
# Build
go build -o nvms ./cmd/nvms

# Probe available tiers
./nvms probe

# Run a sandbox
./nvms run --tier 2 --image ubuntu:latest --name my-sandbox

# List running sandboxes
./nvms list

# Stop a sandbox
./nvms stop --name my-sandbox
```

## Architecture

NanoVMS uses a hexagonal architecture with pluggable adapters:

```
cmd/nvms          CLI entry point
pkg/orchestrate   Orchestration engine (tier selection, lifecycle)
pkg/runtime       Tier registry (30 tiers)
pkg/tier          Tier policy engine
pkg/config        YAML configuration
internal/
  adapters/       Platform-specific adapters (linux, mac, windows, sandbox)
  domain/         Domain types (Sandbox, SandboxConfig)
  ports/          Interface definitions (SandboxPort, TierPort)
```

## Tiers

| Tier | Name | Isolation Level | Status |
|------|------|----------------|--------|
| 1 | WASM | Process + WASM runtime | Partial |
| 2 | gVisor | Container (runsc) | Real |
| 3 | Firecracker | MicroVM | Real |
| 4-10 | Various | Container/VM variants | Stub |
| 11-20 | Specialized | SEV, TDX, Confidential | Stub |
| 21-30 | Enterprise | NFV, Edge, HPC | Stub |

## Configuration

Create `~/.config/nanovms/config.yaml`:

```yaml
log_level: info
default_tier: 2
max_concurrent: 10
allowed_tiers: [1, 2, 3]
```

## License

See LICENSE file for details.

This repository has been archived. All work has been migrated to the Phenotype ecosystem repositories.

For questions or access to migrated code, please contact the repository owner.
