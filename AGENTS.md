# AGENTS.md — NanoVMS

## Project Overview
- **Name**: NanoVMS (Nano Virtual Machine Services)
- **Description**: Go-based runtime and CLI for 3-tier isolation: WASM, gVisor, and Firecracker
- **Location**: `/Users/kooshapari/CodeProjects/Phenotype/repos/nanovms`
- **Language Stack**: Go 1.23+; Node.js only for VitePress docs tooling
- **Published**: Private (Phenotype org)

## Repository Structure
- `api/` — API contracts and generated surfaces
- `docs/` — VitePress docs and reference material
- `go/` — Go runtime and library code
- `sdk/` — shared SDK/client helpers
- `scripts/` — repository automation
- `tests/` — test fixtures and integration coverage
- `.github/workflows/` — CI and security workflows
- `package.json` — docs tooling and VitePress scripts
- `go.mod` — Go module definition

## Quality Checks

From the repository root:
```bash
go fmt ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...
go build ./...
npm run docs:build
```

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

---

For broader policy, use the canonical sources referenced by the parent Claude files.
