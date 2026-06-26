# NanoVMS Project Constitution

> Auto-provisioned by compute/infra DAG Phase E rollout
> Date: 2026-06-25
> Pattern: mirrors `_cu_audit/templates/.kittify/memory/constitution.md`

## Purpose

This constitution captures the technical standards, code quality expectations,
tribal knowledge, and governance rules for nanovms. All features and pull
requests should align with these principles.

## Technical Standards

### Language and Toolchain

- **Primary language**: Rust (see `Cargo.toml` and `rust-toolchain.toml`)
  - Channel: stable
- **Formatter**: `rustfmt` (see `rustfmt.toml`)
  - Strict, enforced via CI
- **Linter**: `clippy`
  - Treat warnings as fatal in CI
- **License audit**: `deny.toml` — `cargo deny check` blocks release

### Governance Gates

See `.github/workflows/` in this repo. Standard gates:

- Tier-0: build, test, fmt, lint
- cargo-deny: supply-chain license/advisory/source audit (added 2026-06-25)
- Scorecard: OpenSSF scorecard automation

### Branch and PR Hygiene

- Branch names: `feat/<slug>`, `fix/<slug>`, `chore/<slug>`
- One logical change per PR
- Reference: compute/infra DAG (phenotype-infra IaC + NVMS integration)
- Reference: phenotype-org-governance ADR-039

## NanoVMS-Specific Knowledge

- NanoVMS is a Firecracker/gVisor/Wasmtime-based NVMS (Nano Virtual Machine
  Sandbox) runtime.
- Designed as part of the PhenoCompose NVMS stack.
- Related repos: PhenoCompose (canonical NVMS driver), phenotype-infra (IaC).
- Uses Cargo workspace pattern.
- CI workflow added 2026-06-25 for cargo-deny block-release.

## Code Quality

- Treat security warnings as fatal
- Run all required tests before claiming work complete
- State what was done, what was not, and why

## Versioning

- SemVer for public crates
- Calendar version for governance documents (YYYY.MM.DD)

## Quick Reference

- Path: always specify exact locations in agent prompts
- Encoding: UTF-8 only
- Context: read what you need, don't re-read unnecessarily
- Quality: secure, tested, documented
- Git: clean commits, descriptive messages
