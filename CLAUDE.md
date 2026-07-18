# CLAUDE.md — nanovms

Extends parent governance. See:
- Global baseline: `~/.claude/CLAUDE.md`
- Phenotype root: `/Users/kooshapari/CodeProjects/Phenotype/repos/CLAUDE.md`
- AgilePlus mandate: `/Users/kooshapari/CodeProjects/Phenotype/repos/AgilePlus`
- Governance reference: `AGENTS.md` (local, this repository)

## Project Overview

**Name:** nanovms (NVMS — NanoVM Service)
**Description:** Unified 3-tier isolation runtime: WASM (~1ms), gVisor (~90ms), Firecracker (~125ms). Merged implementation: KooshaPari/nanovms + BytePort/nvms + PhenoCompose driver.
**Language Stack:** Go, TypeScript
**Location:** `repos/nanovms`
**Status:** Active

## AgilePlus Mandate

All work MUST be tracked in AgilePlus:
- CLI: `cd /Users/kooshapari/CodeProjects/Phenotype/repos/AgilePlus && agileplus <command>`
- Check for existing specs before implementing
- Create spec for new work: `agileplus specify --title "<feature>" --description "<desc>"`
- No code without corresponding AgilePlus spec

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    UNIFIED NVMS STACK                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │ PhenoCompose│    │   NVMS CLI  │    │  BytePort   │    │
│  │   (Rust)    │    │    (Go)     │    │   (Go)      │    │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    │
│         └──────────────────┴──────────────────┘            │
│                            │                                │
│                    ┌───────▼───────┐                        │
│                    │   NVMS Core   │                        │
│                    │    (Merged)   │                        │
│                    └───────┬───────┘                        │
│         ┌──────────────────┼──────────────────┐            │
│         ▼                  ▼                  ▼            │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐        │
│  │    WASM    │    │   gVisor   │    │Firecracker│        │
│  │  (~1ms)    │    │  (~90ms)   │    │ (~125ms)   │        │
│  └────────────┘    └────────────┘    └────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Platform Support

| Platform | Tier 1 (WASM) | Tier 2 (gVisor) | Tier 3 (Firecracker) |
|----------|---------------|-----------------|----------------------|
| macOS    | Native        | Lima/VZ         | Virtualization.framework |
| Linux    | Native        | Native          | KVM                  |
| Windows  | Native        | WSL2            | WSL2                 |

## Quality Checks

From this repository root:
```bash
# Linting and formatting
go fmt ./... && go vet ./...

# Testing
go test ./...

# Build
go build ./cmd/nvms
```

## Worktree & Git Discipline

- Feature work uses repo-specific worktrees: `repos/[PROJECT]-wtrees/<topic>/`
- Canonical repo stays on `main` except during explicit merge operations
- All feature branches are temporary; integrate via pull request or squash commit

## Related Documents

- `SPEC.md` — Comprehensive specification (SOTA research, 3-tier architecture)
- `PLAN.md` — Implementation plan (Phase 1-6)
- `ADR.md` — Architecture decision records
- `README.md` — Project overview and quick start
- `AGENTS.md` — AI agent instructions
- `CHANGELOG.md` — Version history

---

For CI, scripting language hierarchy, and other policies, see the canonical sources listed above.
