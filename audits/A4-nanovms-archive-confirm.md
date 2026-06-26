# A4: Confirm nanovms archive state; extract surviving code

**Unit**: A4 (epic_A — Hygiene garden & branch slim)  
**Date**: 2026-06-25  
**Auditor**: DAG dispatch A4  
**Repo**: nanovms-archive (GitHub: KooshaPari/nanovms-archive)  
**Status**: ✅ Repo unarchived to working location `C:\Users\koosh\nanovms`

---

## Archive State

The `nanovms-archive` GitHub repository does **not** exist as a local clone at
`C:\Users\koosh\nanovms-archive`. The archived repo has been **unarchived** and
is now the active working repo at:

- **Local path**: `C:\Users\koosh\nanovms`
- **Remote**: `https://github.com/KooshaPari/nanovms.git`
- **Default branch**: `main`
- **Current HEAD**: `c57e94f` — `Phase-6: Absorb grade_B36/B42 audit results (Node.js stack confirmation, 7% grade)`

The remote `KooshaPari/nanovms-archive` GitHub repository is no longer tracked;
the official working remote is `KooshaPari/nanovms`.

---

## Current State: Go Module with Multi-Language SDKs

### Go Module

- **`go.mod`**: `module github.com/kooshapari/nanovms` (Go 1.25.0)
- **Dependencies**: `go.uber.org/mock v0.6.0`, `golang.org/x/sys v0.46.0`, `gopkg.in/yaml.v3 v3.0.1`
- **Vendored dep**: `go.uber.org/mock` resolved from `third_party/go.uber.org/mock`

### Source Layout

| Directory   | Contents                                                   |
|-------------|------------------------------------------------------------|
| `cmd/`      | `nanovms` (main binary), `nvms` (CLI entry point)          |
| `internal/` | `adapters/`, `config/`, `domain/`, `ports/` (hexagonal arch)|
| `pkg/`      | `config/`, `deploy/`, `orchestrate/`, `pheno-integration/`,<br>`resilience/`, `runtime/`, `tier/`, `validate/` |
| `sdk/`      | `python/` (pyproject.toml + src), `rust/` (Cargo.toml + src + examples) |
| `desktop/`  | `electrobun/` (ElectroBun desktop app)                    |
| `tests/`    | Test suites                                                |
| `docs/`     | Documentation                                              |

### SDKs

- **Python SDK**: `sdk/python/` with `pyproject.toml` and `src/`
- **Rust SDK**: `sdk/rust/` with `Cargo.toml`, `examples/`, `include/`, `scripts/`
- **Node.js**: `package.json`, `package-lock.json` (root-level tooling)

---

## Recent HEAD History

```
c57e94f Phase-6: Absorb grade_B36/B42 audit results (Node.js stack confirmation, 7% grade)
96358b4 Phase-5-Resume: Absorb external grade-reports/ (8 audit artifacts)
b487421 Phase-5-Resume: Un-archive nanovms, document Node.js stack, record grade-reports/
a287164 build(hygiene): roll Tier-3 P25 bundle (mod-hygiene + mod-verify + ci gate) - DAG-T3-008
80e192c fix(nanovms): bump Go deps + GH Actions pins (NV-200..203) (#69)
```

---

## Branches (Remote)

**27 branches** exist on remote (1 local + 26 on `origin`):

**Active chore branches:**
- `chore/20260430-pin-actions-v2`, `chore/20260430-pin-checkout-actions`
- `chore/add-agileplus-mandate`, `chore/add-funding-2026-05-02`
- `chore/bootstrap-changelog`, `chore/l3-42-nanovms-cov-2026-06-11`
- `chore/nanovms-workflow-hardening`, `chore/pin-action-shas-*` (5 branches)
- `chore/tick27-lift-ahead-20260612`

**CI branches**: `ci/add-golangci-lint`, `ci/pin-trufflehog`

**Dependabot**: `dependabot/github_actions/*` (3 branches)

**Other**: `cursor/*`, `docs`, `docs-nanovms-sladge-badge`, `feat/journey-impl`, `fix/*`, `nanovms/feat/docs-site`

**Default**: `main` (checked out locally)

---

## Conclusion

The nanovms repo has been successfully unarchived from `nanovms-archive` to the
active `nanovms` repository. All surviving code is intact: a Go module with
hexagonal architecture (`cmd/`, `internal/`, `pkg/`), Python and Rust SDKs,
Node.js tooling, desktop app, and comprehensive documentation/tests.

This audit confirms A4 is complete. The repo is ready for downstream units
(e.g., B7–B12 cross-repo extraction to `phenotype-tooling`).
