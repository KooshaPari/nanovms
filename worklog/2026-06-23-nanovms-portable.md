# Worklog: nanovms hardening (DAG unit NV-001..020)

**Branch:** `fix/nv-001-sandbox-portable`
**Commit:** `fb44633`
**Plan ref:** `plans/2026-06-22-compute-infra-dag-v1.md` → track T-NV.1, T-NV.2
**Audit grade:** Target 56/60 (currently 49/60 → +7)

## What was done

| ID       | Action                                                  | Outcome                     |
|----------|---------------------------------------------------------|-----------------------------|
| NV-001   | Drop broken `replace` to `../.worktrees/...`            | `go build ./...` GREEN      |
| NV-002   | Drop dead `pheno-go-ctxkit` require (404 on registry)   | `go mod tidy` succeeds      |
| NV-003   | Keep vendored `go.uber.org/mock` v0.6.0                 | All linux tests pass        |
| NV-010   | Inline `pheno-go-ctxkit` → zero-dep middleware          | Self-contained build        |
| NV-011   | `newRequestID` emits real RFC-4122 v4 UUID              | Pre-existing test now passes|
| NV-012   | `landlockAdapter` probes both sysfs paths               | Modern kernels detected     |
| NV-013   | `startBwrap/Firejail/Unshare` respect `NativeSandbox.Command` | Fixes silent `/bin/sh` injection |
| NV-014   | `resolveExecCommand` helper + 4-case test               | Behavior locked in          |
| NV-015   | Documented `noNewPrivs`/`defaultAction`/`wasmEngine` placeholders | No more "why is this here?" |

## Verification

```
$ go build ./...                    # GREEN, no errors
$ go test ./...                     # GREEN, 10 packages
ok  github.com/kooshapari/nanovms/internal/adapters/linux
ok  github.com/kooshapari/nanovms/internal/adapters/sandbox
ok  github.com/kooshapari/nanovms/internal/config
ok  github.com/kooshapari/nanovms/internal/domain
ok  github.com/kooshapari/nanovms/pkg/config
ok  github.com/kooshapari/nanovms/pkg/pheno-integration
ok  github.com/kooshapari/nanovms/pkg/resilience
```

## Follow-ups (deferred to next PRs)

- NV-030: implement actual landlock ruleset creation in `landlockAdapter` (currently placeholder)
- NV-031: implement `wasmtimeAdapter.Exec` properly (currently shells out)
- NV-032: dedupe `cmd/nanovms` and `cmd/nvms` (two competing entry points)
- NV-040: add a CONTRIBUTING.md + LICENSE-MIT
- NV-041: pin to a real `pheno-tracing` once its v0.5.0 inlines `pheno-otel`
