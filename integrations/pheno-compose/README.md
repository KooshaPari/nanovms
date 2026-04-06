# PhenoCompose NVMS Driver

Integration layer between PhenoCompose and NVMS.

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
