// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"context"
	"strings"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

// ToolkitResolver discovers installed CUDA toolkits using environment precedence.
type ToolkitResolver struct {
	Inspector HostToolkitInspector
}

// Resolve discovers one acceptable toolkit version for a profile.
func (resolver ToolkitResolver) Resolve(ctx context.Context, request Request, profile LockedProfile) (ResolvedToolkit, error) {
	if resolver.Inspector.Runner == nil {
		return ResolvedToolkit{}, providerError(CodeToolkitNotFound, "toolkit command runner is required")
	}
	inspector := resolver.Inspector
	if request.ToolkitRoot != "" && inspector.ToolkitRoot == "" {
		inspector.ToolkitRoot = request.ToolkitRoot
	}
	var lastErr error
	for _, version := range profile.ToolkitCandidates() {
		toolkit, err := inspector.ResolveToolkit(ctx, toolkitDiscoveryRequest(request, version))
		if err != nil {
			lastErr = err
			continue
		}
		if profile.AcceptsToolkitVersion(toolkit.Version) {
			return toolkit, nil
		}
		lastErr = providerError(CodeToolkitVersionMismatch,
			"observed CUDA toolkit %s is not accepted for profile %q", toolkit.Version, profile.ID)
	}
	if lastErr == nil {
		lastErr = providerError(CodeToolkitNotFound, "CUDA toolkit for profile %q was not found", profile.ID)
	}
	return ResolvedToolkit{}, lastErr
}

func toolkitDiscoveryRequest(request Request, version string) ToolkitDiscoveryRequest {
	// WSLDistribution on Request drives read-only WSL metadata in buildContract;
	// toolkit discovery stays on the host unless the caller supplies WSL-scoped
	// environment overrides for CUDA paths.
	return ToolkitDiscoveryRequest{
		Environment: request.Environment,
		CUDAToolkit: version,
	}
}

func plannedMutations(toolkit ResolvedToolkit) []Mutation {
	if toolkit.Root == "" {
		return nil
	}
	versionKey := "CUDA_PATH_V" + strings.NewReplacer(".", "_", "-", "_").Replace(toolkit.Version)
	mutations := []Mutation{
		{Kind: "set_env", Key: "NANOVMS_CUDA_TOOLKIT_ROOT", Value: toolkit.Root},
		{Kind: "set_env", Key: "CUDA_PATH", Value: toolkit.Root},
		{Kind: "set_env", Key: versionKey, Value: toolkit.Root},
	}
	return mutations
}

func toolkitRecord(profile LockedProfile, toolkit ResolvedToolkit) ToolkitRecord {
	return ToolkitRecord{
		Requested:  profile.CUDAToolkitTarget,
		Observed:   toolkit.Version,
		Root:       toolkit.Root,
		Executable: toolkit.Executable,
	}
}

func compatibilityRecord(profile LockedProfile, device gpu.Device, toolkit ResolvedToolkit, compatible bool) CompatibilityRecord {
	return CompatibilityRecord{
		CUDAToolkitRequested: profile.CUDAToolkitTarget,
		CUDAToolkitObserved:  toolkit.Version,
		DriverVersion:        device.DriverVersion,
		DriverCUDACeiling:    device.DriverCUDACeiling,
		ComputeCapability:    device.ComputeCapability,
		Compatible:           compatible,
	}
}

func validateCompatibility(device gpu.Device, profile LockedProfile, toolkit ResolvedToolkit) error {
	if !profile.AcceptsToolkitVersion(toolkit.Version) {
		return providerError(CodeToolkitVersionMismatch,
			"observed CUDA toolkit %s is not accepted for profile %q", toolkit.Version, profile.ID)
	}
	return gpu.ValidateCompatibility(device, profile.artifactRequirements(toolkit.Version))
}
