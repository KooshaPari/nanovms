# WP-SHARED-001: phenotype-shared → phenotype-manifest boundary spike

## Background

`phenotype-shared` was designed as a shared workspace for cross-repo primitives.
After audit, only `phenotype-manifest` has a real boundary. The other 3 components
(`ffi_utils`, `phenotype-port-adapter-shim`, `phenotype_shared` Python) have zero
active consumers and should be removed.

## Scope

### Delete (zero consumers, trivially reproducible)
- `crates/ffi_utils/` — 19-line type alias, broken path dep in helios-cli
- `crates/phenotype-port-adapter-shim/` — 321 lines, competing impl in OmniRoute
- `phenotype_shared/` — Python stdlib wrappers, zero imports
- `pyproject.toml` — Python package config
- `tests/test_shared.py` — Python tests
- `.benchmarks/`, `.hypothesis/`, `.pytest_cache/` — stale tooling artifacts

### Keep (strong boundary, production-ready)
- `crates/phenotype-manifest/` — 277 LOC, 9 integration tests + doc-tests
- `schemas/odin.nvms.schema.json` — polyglot boundary artifact (Rust → Go/TS)
- `Cargo.toml` (workspace root) — trim to manifest-only
- `README.md` — update to reflect manifest-only scope

### Boundary spike goal
Wire BytePort's Go Provisioner to validate `odin.nvms` manifests against
`schemas/odin.nvms.schema.json` using `gojsonschema`. This proves the
polyglot shared-schema pattern end-to-end.

## Effort
- Cleanup: ~1 hour (delete dead code, update workspace)
- Spike: ~2-4 hours (Go Provisioner integration)
