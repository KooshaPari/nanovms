# SPEC.md — nanovms

## Architecture Overview

nanovms is a Go CLI for lightweight VM deployment and management, with Rust integration for low-level system operations. It hosts three main packages: `sdk`, `virter`, and `phylmnt` for VM lifecycle management.

## Stack

- **Languages:** Go (primary), Rust (integration), C/C++ (low-level bindings)
- **Framework:** Go standard library (net/http, os/exec, etc.), CGo (for C/C++ interop)
- **Key Libraries:** Go modules (no external framework), tokio (Rust side)
- **Build System:** Go + Cargo + Justfile
- **Package Manager:** Go modules (Go), Cargo (Rust)

## Key Commands

| Command | Description |
|---------|-------------|
| `just build` | Compile all packages (Go + Rust) |
| `just test` | Run all tests (go test + cargo test) |
| `just lint` | Run go vet + cargo clippy |
| `just run` | Execute the application |
| `just check` | Quick check for compilation errors |

## Design Decisions

- **Go + Rust + C/C++ polyglot architecture:** Go handles high-level CLI and orchestration, Rust handles low-level system operations, and C/C++ provides native bindings.
- **Multi-subpackage organization:** The repository is organized into three main packages (`sdk`, `virter`, `phylmnt`) for clear separation of VM lifecycle concerns.
- **Justfile for build orchestration:** All build, test, lint, and run commands are centralized in a Justfile for consistent developer workflow.

## Integration Points

- No external `pheno-*` crates are integrated; all dependencies are standard library and local modules.

## Repository

- **Path:** `/Users/kooshapari/CodeProjects/Phenotype/repos/nanovms`
- **Repository:** `nanovms`
- **License:** MIT
- **Version:** 0.1.0
