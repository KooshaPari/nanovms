# Evaluation action contract

Version: `nanovms.io/evaluation-action/v1`

CLI: `nvms action --request -` reads one JSON request from stdin and always writes
one `EvaluationResult` JSON object to stdout. On failure, stderr also prints
`nvms action: <error_code>: <message>`.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage |
| 3 | Invalid JSON |
| 4 | Invalid request / validation (`invalid_*`, `adapter_unconfigured`, `provider_rejected`, …) |
| 5 | Result encode failure |
| 6 | Contention (`reservation_failed`, `output_lock_failed`) |
| 7 | Host probe (`inventory_*`, `toolkit_*`, `inspection_*`) |
| 8 | Action runtime (`action_failed`, `action_timeout`, `output_truncated`) |
| 9 | Evidence / cleanup (`cleanup_failed`, `job_lock_mismatch`, `output_root_*`, …) |

Fine-grained diagnosis always uses JSON `error_code`. Process exits are a coarse agent branch.

## Stable error codes

Catalog source of truth: `pkg/orchestrate/error_codes.go` (`AllEvaluationErrorCodes`).

| Code | Agent action |
|------|----------------|
| `invalid_request` / `invalid_output_root` / `invalid_reservation_path` / `invalid_limits` / `invalid_manifest_digest` / `invalid_engine_token` / `invalid_podman_pipe` | Fix request / store path bind |
| `adapter_unconfigured` | Wire registry, inventory, inspector, runner, reservations |
| `provider_rejected` / `fallback_rejected` | Use exact Podman backend; no fallbacks |
| `gpu_binding_rejected` | Fix GPU bindings vs resource manifest |
| `inventory_unavailable` / `inventory_mismatch` | Fix host GPU inventory |
| `toolkit_rejected` / `toolkit_not_found` / `toolkit_ambiguous` / `toolkit_version_mismatch` / `toolkit_inspection_timeout` | Fix CUDA toolkit discovery / compatibility |
| `inspection_failed` / `inspection_mismatch` | Fix Podman/CDI/host probes |
| `output_root_create_failed` / `output_root_collision` / `output_root_space_failed` / `output_lock_failed` | Fix output path, space, or lock contention (backoff) |
| `reservation_failed` | Backoff; another evaluation holds GPUs |
| `output_snapshot_failed` | Inspect output_root readability |
| `action_timeout` / `output_truncated` / `action_failed` | Retry or fix workload (these beat job-output codes when the command failed) |
| `ambiguous_job_output` / `job_lock_mismatch` | Inspect Harbor job evidence (`candidate_job_directories`) |
| `cleanup_failed` | Primary action may have succeeded; release/cleanup failed |
| `output_root_cleanup_failed` | Created root preserved because cleanup was unsafe |
| `evaluation_failed` | Untyped failure fallback |

## Ordering note

GPU reservation is acquired after inventory and before inspection/run so probes cannot
race another evaluation for the same UUIDs. Inspect failures still release the lease.

## Provenance fields (additive)

- `output_root` / `output_lock_path` / `coordinator_lock_path` — paths for FS and lock failures
- `reservation_path` / `reservation_token` / `reservation_expires_at` — lease evidence when acquired
- `lock_invocation` — echoed request lock invocation
- `command_sha256` — fingerprint of the launched executable+argv (after WSL wrapping)
- `candidate_job_directories` — set when job-dir counting is ambiguous
- `output_root_created` — this attempt created the directory (cleared if later removed)
- `output_root_available_bytes` — present only when space probe succeeded
- `output_root_cleanup` — `removed` or `preserved` when pre-action cleanup ran
- `toolkit_*` — empty paths mean precompiled artifact did not require host nvcc
- `job_directory` — Harbor job dir when exactly one new directory appears
