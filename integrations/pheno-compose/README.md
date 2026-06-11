# PhenoCompose NVMS Driver

Integration layer between PhenoCompose and NVMS.

## Overview

The PhenoCompose NVMS Driver bridges the Rust-based PhenoCompose orchestrator
with the Go-based NanoVMS runtime. It exposes a unified interface for deploying
workloads across NanoVMS's three isolation tiers.

## Architecture

```
PhenoCompose (Rust) → NVMS Driver → NanoVMS (Go)
                              ↓
                    ┌─────────┴─────────┐
                    │   3-Tier Isolation │
                    ├─────────┬─────────┤
                    │ WASM    │ ~1ms    │
                    │ gVisor  │ ~90ms   │
                    │ Firecracker│~125ms│
                    └─────────┴─────────┘
```

## Installation

```bash
# Install the PhenoCompose CLI with NVMS support
cargo install pheno-compose --features nvms-driver

# Or add to your Cargo.toml
[dependencies]
pheno-compose = { version = "0.1", features = ["nvms-driver"] }
```

## Usage

```rust
use pheno_compose::PhenoCompose;

// PhenoCompose uses NVMS as primary runtime
let compose = PhenoCompose::from_file("nvms.yaml")?;

// NVMS provides tiered isolation
compose.deploy_tier1_wasm()?;      // Fast, trusted
compose.deploy_tier2_gvisor()?;    // Browser automation
compose.deploy_tier3_firecracker()?; // Full isolation
```

## Configuration

Example `nvms.yaml` for PhenoCompose:

```yaml
runtime: nvms
tier: 2
resources:
  cpu: 2
  memory: "1G"
  disk: "10G"
image: "ubuntu-22.04"
network:
  type: nat
  subnet: "192.168.100.0/24"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NVMS_DRIVER_ENDPOINT` | `http://localhost:8080` | NanoVMS API endpoint |
| `NVMS_DRIVER_TIER` | `1` | Default isolation tier |
| `NVMS_DRIVER_TIMEOUT` | `30` | Request timeout in seconds |

## Testing

```bash
# Run the PhenoCompose NVMS test suite
cargo test --features nvms-driver

# Run integration tests against a local NanoVMS daemon
NVMS_DRIVER_ENDPOINT=http://localhost:8080 cargo test --test integration
```

## License

Apache-2.0
