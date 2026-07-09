# ADR 0002: Sandbox Adapter Decomposition

## Status

Accepted. 2026-07-09.

## Context

`internal/adapters/sandbox/sandbox.go` had grown to 1040 lines (originally), exceeding the AGENTS.md hard limit of 500 lines per module.

## Decision

Decompose `sandbox.go` into 8 cohesive files:

- `sandbox.go` (350) - package + imports + Adapter struct + Adapter methods
- `adapter.go` - Adapter facade methods (Create/Start/Stop/Delete/ListRuntimes)
- `gvisor.go` - gVisor (runsc) adapter
- `landlock.go` - Linux Landlock adapter
- `seccomp.go` - seccomp adapter
- `wasmtime.go` - wasmtime adapter
- `helpers.go` - utility functions (logs, exec, metrics, ID generation)
- `native.go` - native adapter (bwrap, firejail, unshare)

And further split the test file into:

- `sandbox_test.go` (9) - just the package decl
- `sandbox_helpers_test.go` (152) - ID generation, status, helpers
- `adapter_test.go` (184) - Adapter facade tests
- `native_test.go` (64) - native adapter tests

## Consequences

- All files now under the 500-line hard limit
- Most under the 350-line target
- Tests still pass (all 24 test functions preserved)
- Public API unchanged
