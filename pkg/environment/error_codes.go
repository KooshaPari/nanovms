// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

// Stable environment error_code values (public CLI contract for nvms environment).
const (
	CodeApplyFailed              = "apply_failed"
	CodeContractDigestFailed     = "contract_digest_failed"
	CodeEnvironmentDrift         = "environment_drift"
	CodeEnvironmentFailed        = "environment_failed"
	CodeEnvironmentNotApplied    = "environment_not_applied"
	CodeGPUNotFound              = "gpu_not_found"
	CodeInvalidProfile           = "invalid_profile"
	CodeInvalidRequest           = "invalid_request"
	CodeInventoryEmpty           = "inventory_empty"
	CodeInventoryFailed          = "inventory_failed"
	CodeToolkitAmbiguous         = "toolkit_ambiguous"
	CodeToolkitInspectionTimeout = "toolkit_inspection_timeout"
	CodeToolkitNotFound          = "toolkit_not_found"
	CodeToolkitVersionMismatch   = "toolkit_version_mismatch"
	CodeWSLInspectionFailed      = "wsl_inspection_failed"
)

// AllEnvironmentErrorCodes returns the frozen catalog.
func AllEnvironmentErrorCodes() []string {
	return []string{
		CodeApplyFailed,
		CodeContractDigestFailed,
		CodeEnvironmentDrift,
		CodeEnvironmentFailed,
		CodeEnvironmentNotApplied,
		CodeGPUNotFound,
		CodeInvalidProfile,
		CodeInvalidRequest,
		CodeInventoryEmpty,
		CodeInventoryFailed,
		CodeToolkitAmbiguous,
		CodeToolkitInspectionTimeout,
		CodeToolkitNotFound,
		CodeToolkitVersionMismatch,
		CodeWSLInspectionFailed,
	}
}

// ProcessExitFor maps environment error codes onto the shared nvms exit classes
// (same numeric taxonomy as orchestrate.ProcessExitFor).
func ProcessExitFor(code string) int {
	switch code {
	case CodeInvalidRequest, CodeInvalidProfile:
		return 4
	case CodeInventoryFailed, CodeInventoryEmpty, CodeGPUNotFound,
		CodeToolkitNotFound, CodeToolkitAmbiguous, CodeToolkitVersionMismatch,
		CodeToolkitInspectionTimeout, CodeWSLInspectionFailed:
		return 7
	case CodeApplyFailed, CodeContractDigestFailed, CodeEnvironmentDrift,
		CodeEnvironmentNotApplied:
		return 9
	default:
		return 4
	}
}
