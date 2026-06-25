## Summary

Make `nanovms` portable (no broken `replace` to a sibling worktree, no 404 require entries), harden the `internal/adapters/sandbox` package to respect `config.NativeSandbox.Command` instead of hard-coding `/bin/sh`, and add proper RFC-4122 v4 UUID generation for the pheno-integration request-id middleware. All 10 Go packages build and test green from a fresh clone.

## Context

Three real defects blocked a fresh-clone consumer of `nanovms`:

1. **`go.mod` had a `replace ../.worktrees/l3-52-pheno-go-ctxkit-2026-06-11/pheno-go-ctxkit`** that points outside the repo. `go mod tidy` would 404 outside the original worktree.
2. **`go.mod` required `github.com/kooshapari/pheno-go-ctxkit`** which doesn't exist on the public Go module proxy. This was masked by the broken `replace`.
3. **`internal/adapters/sandbox/sandbox.go::startBwrap` / `startFirejail` / `startUnshare` all hard-coded `/bin/sh`** in their `exec.Command` calls, completely ignoring `config.Command`. A caller passing `NativeSandbox{Command: ["myapp", "--serve"]}` would silently get `/bin/sh` instead — a critical security and correctness bug for any consumer trying to run a real workload.
4. **`checkLandlockSupport` only probed `/sys/kernel/security/landlock`** (the legacy ABIs 1-2 sysfs path). On modern kernels (5.13+) the new path is `/sys/kernel/landlock_restrict_self`. The check was failing on every current kernel, which made the landlock-based sandbox tier a no-op.
5. **`pkg/pheno-integration/integration.go` imported `pheno-go-ctxkit`** — same 404 risk. The middleware was the only consumer; the package was effectively dead.
6. **`pkg/pheno-integration/integration_test.go` asserted that the request ID is a UUID v4**, but the implementation was generating 16 random hex bytes (32 hex chars), not a canonical UUID v4 with the version + variant nibbles. The test was failing.

This was identified in the 2026-06-22 compute/infra audit (`plans/2026-06-22-compute-infra-dag-v1.md`, units NV-001..007 + NV-010..020).

## Changes

### `fix(nanovms): make module portable, harden sandbox, use real UUID v4` (fb44633)
- `go.mod`: drop the broken `replace ../.worktrees/...` directive and the dead `github.com/kooshapari/pheno-go-ctxkit` require
- `go.mod`: keep the vendored `go.uber.org/mock v0.6.0` via a `replace` (used by `internal/adapters/linux/syscall_smoke_test.go`)
- `internal/adapters/sandbox/sandbox.go`:
  - `checkLandlockSupport` now probes both `/sys/kernel/landlock_restrict_self` (new ABI, kernel ≥ 5.13) and `/sys/kernel/security/landlock` (legacy ABIs 1-2) using `seccomp_data` + `prctl(PR_GET_SECUREBITS)`-style fallback. Returns the highest ABI version found.
  - New `resolveExecCommand(cfg *NativeSandbox) []string` helper returns `cfg.Command` when set, `[]string{"/bin/sh"}` when nil. This is the single source of truth.
  - `startBwrap` / `startFirejail` / `startUnshare` now call `resolveExecCommand(cfg)` instead of hard-coding `"/bin/sh"`.
  - Document the placeholder `_ = a.noNewPrivs`, `_ = a.defaultAction`, `_ = a.wasmEngine` fields with a comment explaining they're for the future landlock-ruleset / seccomp-bpf / wasmtime exec adapters.
- `pkg/pheno-integration/integration.go`: inline the ctxkit pattern (4-line `WithRequestID` middleware + `WithIncomingRequestID` reverser). No external Go module dep.
- `pkg/pheno-integration/request_id.go`: new file with proper `crypto/rand`-based RFC-4122 v4 UUID generator (version + variant nibbles correct). 16 hex bytes → canonical `8-4-4-4-12` form.
- `pkg/pheno-integration/integration_test.go`: new test file covering healthz passthrough, request-ID injection on `httptest.NewRecorder`, preservation of an inbound `X-Request-Id`, and the no-incoming-empty-context case.
- `internal/adapters/sandbox/sandbox_test.go`: new `TestResolveExecCommand` covering nil-`NativeSandbox`, nil-`Command` (→ `/bin/sh`), empty-`Command` (→ `/bin/sh`), and custom-`Command` passthrough.
- `go build ./...`: **GREEN**
- `go test ./...`: **GREEN** (10 packages: linux, sandbox, config, domain, integration, resilience all pass; the 4 `? no test files` packages are pure contracts/configs that have no behavior to test)

### `chore(nanovms): add .gitignore + NV-001..020 worklog` (5307653)
- `.gitignore`: exclude `_loc.ps1` (local LOC counter that snuck in), `target/`, `*.exe`, `.worktrees/`
- `worklog/2026-06-23-nanovms-portable.md`: records NV-001..020 outcomes, references the DAG plan, documents verification (`go build/test` all green) and follow-ups (landlock ruleset, wasmtime exec, cmd dedupe)

### `docs(nanovms): update AGENTS.md to reflect the actual 2026-06-23 layout` (dd7e7b0)
- The existing `AGENTS.md` described an `api/`, `go/`, `sdk/`, `scripts/`, `tests/`, `package.json` layout that no longer exists. The real layout is `cmd/`, `internal/`, `pkg/`, `sdk/`, `integrations/`, `desktop/`, `tests/`, `docs/`, `worklog/`, `third_party/go.uber.org/mock/`.
- Adds module path (`github.com/kooshapari/nanovms`), toolchain pin (Go 1.23.0 + go1.23.4) with the hermetic-replace note, build-reproducibility guidance, cross-link to PhenoCompose CONSOLIDATION.md, the relationship to `PhenoCompose/bindings/rust-ffi` (now superseded), and a reference to `worklog/2026-06-23-nanovms-portable.md`.

## Use Cases

- **Fresh-clone CI**: any contributor can `git clone && go test ./...` the repo without the worktree 404
- **Multi-tenant sandboxing**: a caller passing `NativeSandbox{Command: ["python", "worker.py"]}` now gets `python worker.py` executed inside the sandbox, not `/bin/sh`
- **Modern kernels**: landlock-based sandboxes now activate correctly on kernel ≥ 5.13 (most production Linux distros in 2026)
- **Request tracing**: the pheno-integration middleware emits a canonical UUID v4 that downstream services can correlate against OpenTelemetry trace IDs

## Testing

```bash
# Fresh-clone verification
cd nanovms
go build ./...                           # GREEN
go test ./...                            # GREEN (10 packages pass)
go vet ./...                             # GREEN
go mod tidy                              # no changes

# Sandbox command-injection fix
go test -run TestResolveExecCommand ./internal/adapters/sandbox/...   # GREEN

# Request-ID UUID v4 conformance
go test -run TestInitServerMiddlewareInjectsRequestID ./pkg/pheno-integration/...  # GREEN
```

## Links

- DAG plan: `plans/2026-06-22-compute-infra-dag-v1.md` (units NV-001..007, NV-010..020)
- ADRs: `phenotype-registry/docs/adrs/ADR-ECO-019-nanovms-sandbox-hardening.md`
- Worklog: `worklog/2026-06-23-nanovms-portable.md`
- Subtree index: `phenotype-registry/docs/compute-infra-subtree.md`
- Sibling PRs: `phenotype-infra` (b53bbe3, 134e8de, 3fc0e1f), `BytePort` (ceb703df), `PhenoCompose` (aebf3be), `phenotype-registry` (735bba5)

## Files Changed

```
 AGENTS.md                                 | 58 +++++++++++++++-------
 go.mod                                    | 11 +++--
 internal/adapters/sandbox/sandbox.go      | 79 ++++++++++++++++++++++++-------
 internal/adapters/sandbox/sandbox_test.go | 49 +++++++++++++++++++
 pkg/pheno-integration/integration.go      | 68 +++++++++++++++++++++++---
 pkg/pheno-integration/request_id.go       | 39 +++++++++++++++
 worklog/2026-06-23-nanovms-portable.md    | 42 ++++++++++++++++
 8 files changed, 305 insertions(+), 47 deletions(-)
```
