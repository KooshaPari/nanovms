# AGENTS.md — NanoVMS

## Project Overview
- **Name**: NanoVMS (Nano Virtual Machine Services)
- **Module**: `github.com/kooshapari/nanovms`
- **Description**: Go-based runtime and CLI for 3-tier isolation: WASM, gVisor,
  and Firecracker. Acts as the native sandbox/VMM layer for the Phenotype
  compute mesh (PhenoCompose driver and phenotype-infra IaC daemons both
  call into it).
- **Language Stack**: Go 1.23 (toolchain 1.23.4); Node.js only for VitePress
  docs tooling
- **Published**: Private (Phenotype org, archived as a public mirror)

## Repository Structure (live, 2026-06-23)
- `cmd/nanovms/` — primary CLI entry point (multi-platform VM orchestrator)
- `cmd/nvms/` — legacy single-tier CLI (kept for backwards compatibility;
  see ADR-035 for deprecation timeline)
- `internal/adapters/` — pluggable backends (`linux` for syscall mocks,
  `sandbox` for bwrap/firejail/unshare landlock, `krun` for libkrun VM)
- `internal/domain/` — core types (`SandboxConfig`, `NativeSandboxType`,
  `SandboxTier`, etc.)
- `internal/ports/` — hexagonal-architecture ports (compiler, runtime, etc.)
- `pkg/pheno-integration/` — Phenotype-flavoured helpers (request-id,
  healthz, ctx propagation) — see `worklog/2026-06-23-nanovms-portable.md`
  for the pheno-go-ctxkit replacement
- `sdk/` — generated client SDKs (kept for downstream consumers)
- `integrations/` — first-party integrations (e.g. cloud-init seeders)
- `desktop/` — desktop helper utilities (Tauri shim, system tray, etc.)
- `third_party/go.uber.org/mock/` — vendored copy of `go.uber.org/mock`
  v0.6.0 (replaced via `go.mod` to keep builds reproducible offline)
- `tests/` — integration tests + fixtures
- `docs/` — VitePress docs (built via `npm run docs:build`)
- `worklog/` — L1/L2/L3 work unit log files (canonical)
- `bindings/` — *historical* (PhenoCompose `bindings/rust-ffi` supersedes
  the in-tree FFI stub; tracked via PhenoCompose `CONSOLIDATION.md`)

## Quality Checks
From the repository root:
```bash
go fmt ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...
go build ./...
```

## Build Reproducibility
- `go.mod` pins `go 1.23.0` + `toolchain go1.23.4`
- `go.uber.org/mock` is replaced with `./third_party/go.uber.org/mock`
  to keep CI hermetic. **Do not** remove this replace directive without
  first checking that the upstream release v0.6.0 is identical to the
  vendored copy (`diff -r` + `go mod why`).

## Worktree & Git Discipline
- Feature work uses repo-specific worktrees: `repos/nanovms-wtrees/<topic>/`
- Keep the canonical repo on `main` except during explicit merge operations
- Use temporary feature branches for implementation work and integrate via PR or squash commit

## CI / Workflow Guidance
- Keep workflow action references pinned and review them when dependencies change
- Prefer Linux runners unless a workflow has a hard macOS requirement
- Keep security workflows in `.github/workflows/` aligned with the current toolchain

## Related Documents
- `README.md` — project overview and quick start
- `CLAUDE.md` — Claude-specific repository guidance
- `SPEC.md` — system specification and architecture notes
- `PLAN.md` — implementation plan
- `ADR.md` — architecture decisions
- `CHANGELOG.md` — version history
- `CONSOLIDATION.md` — historical consolidation audit (now lives in
  `PhenoCompose/CONSOLIDATION.md`; this repo is the upstream source for
  items marked as migrated in the PhenoCompose audit)

---

For broader policy, use the canonical sources referenced by the parent Claude files.
