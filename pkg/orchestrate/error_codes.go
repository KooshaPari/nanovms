// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

// Stable evaluation error_code values. These are a public CLI/agent contract;
// keep EVALUATION.md and ProcessExitFor in sync when adding codes.
const (
	CodeActionFailed            = "action_failed"
	CodeActionTimeout           = "action_timeout"
	CodeAdapterUnconfigured     = "adapter_unconfigured"
	CodeAmbiguousJobOutput      = "ambiguous_job_output"
	CodeCleanupFailed           = "cleanup_failed"
	CodeEvaluationFailed        = "evaluation_failed"
	CodeFallbackRejected        = "fallback_rejected"
	CodeGPUBindingRejected      = "gpu_binding_rejected"
	CodeInspectionFailed        = "inspection_failed"
	CodeInspectionMismatch      = "inspection_mismatch"
	CodeInvalidEngineToken      = "invalid_engine_token"
	CodeInvalidLimits           = "invalid_limits"
	CodeInvalidManifestDigest   = "invalid_manifest_digest"
	CodeInvalidOutputRoot       = "invalid_output_root"
	CodeInvalidPodmanPipe       = "invalid_podman_pipe"
	CodeInvalidRequest          = "invalid_request"
	CodeInvalidReservationPath  = "invalid_reservation_path"
	CodeInventoryMismatch       = "inventory_mismatch"
	CodeInventoryUnavailable    = "inventory_unavailable"
	CodeJobLockMismatch         = "job_lock_mismatch"
	CodeOutputLockFailed        = "output_lock_failed"
	CodeOutputRootCleanupFailed = "output_root_cleanup_failed"
	CodeOutputRootCollision     = "output_root_collision"
	CodeOutputRootCreateFailed  = "output_root_create_failed"
	CodeOutputRootSpaceFailed   = "output_root_space_failed"
	CodeOutputSnapshotFailed    = "output_snapshot_failed"
	CodeOutputTruncated         = "output_truncated"
	CodeProviderRejected        = "provider_rejected"
	CodeReservationFailed       = "reservation_failed"
	CodeToolkitAmbiguous        = "toolkit_ambiguous"
	CodeToolkitInspectionTO     = "toolkit_inspection_timeout"
	CodeToolkitNotFound         = "toolkit_not_found"
	CodeToolkitRejected         = "toolkit_rejected"
	CodeToolkitVersionMismatch  = "toolkit_version_mismatch"
)

// AllEvaluationErrorCodes returns the frozen catalog used by docs and CI.
func AllEvaluationErrorCodes() []string {
	return []string{
		CodeActionFailed,
		CodeActionTimeout,
		CodeAdapterUnconfigured,
		CodeAmbiguousJobOutput,
		CodeCleanupFailed,
		CodeEvaluationFailed,
		CodeFallbackRejected,
		CodeGPUBindingRejected,
		CodeInspectionFailed,
		CodeInspectionMismatch,
		CodeInvalidEngineToken,
		CodeInvalidLimits,
		CodeInvalidManifestDigest,
		CodeInvalidOutputRoot,
		CodeInvalidPodmanPipe,
		CodeInvalidRequest,
		CodeInvalidReservationPath,
		CodeInventoryMismatch,
		CodeInventoryUnavailable,
		CodeJobLockMismatch,
		CodeOutputLockFailed,
		CodeOutputRootCleanupFailed,
		CodeOutputRootCollision,
		CodeOutputRootCreateFailed,
		CodeOutputRootSpaceFailed,
		CodeOutputSnapshotFailed,
		CodeOutputTruncated,
		CodeProviderRejected,
		CodeReservationFailed,
		CodeToolkitAmbiguous,
		CodeToolkitInspectionTO,
		CodeToolkitNotFound,
		CodeToolkitRejected,
		CodeToolkitVersionMismatch,
	}
}

// Process exit classes for nvms action / environment (JSON error_code stays fine-grained).
const (
	ExitOK             = 0
	ExitUsage          = 2
	ExitInvalidJSON    = 3
	ExitInvalidRequest = 4
	ExitEncodeFailure  = 5
	ExitContention     = 6
	ExitHostProbe      = 7
	ExitActionRuntime  = 8
	ExitEvidence       = 9
)

// ProcessExitFor maps a stable error_code to a process exit class.
func ProcessExitFor(code string) int {
	switch code {
	case CodeInvalidRequest, CodeInvalidOutputRoot, CodeInvalidReservationPath,
		CodeInvalidLimits, CodeInvalidManifestDigest, CodeInvalidEngineToken,
		CodeInvalidPodmanPipe, CodeAdapterUnconfigured, CodeProviderRejected,
		CodeFallbackRejected, CodeGPUBindingRejected:
		return ExitInvalidRequest
	case CodeReservationFailed, CodeOutputLockFailed:
		return ExitContention
	case CodeInventoryUnavailable, CodeInventoryMismatch, CodeInspectionFailed,
		CodeInspectionMismatch, CodeToolkitRejected, CodeToolkitNotFound,
		CodeToolkitAmbiguous, CodeToolkitVersionMismatch, CodeToolkitInspectionTO:
		return ExitHostProbe
	case CodeActionFailed, CodeActionTimeout, CodeOutputTruncated:
		return ExitActionRuntime
	case CodeCleanupFailed, CodeOutputRootCleanupFailed, CodeAmbiguousJobOutput,
		CodeJobLockMismatch, CodeOutputSnapshotFailed, CodeOutputRootCreateFailed,
		CodeOutputRootCollision, CodeOutputRootSpaceFailed:
		return ExitEvidence
	default:
		return ExitInvalidRequest
	}
}
