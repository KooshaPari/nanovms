// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"github.com/kooshapari/nanovms/pkg/gpu"
)

const (
	pythonInterpreterPlaceholder = "python3"
)

var (
	// AmpereGPUUUID is the lockable RTX 3090 Ti profile identity.
	AmpereGPUUUID = gpu.UUID("GPU-8d337a84-43de-158d-7526-7175288a6064")
	// PascalGPUUUID is the lockable GTX 1080 Ti profile identity.
	PascalGPUUUID = gpu.UUID("GPU-e1e28703-34bd-e46b-6dc1-d7c39fa5612c")
)

// LockedProfile is an immutable environment profile definition.
type LockedProfile struct {
	ID                      ProfileID
	GPUUUID                 gpu.UUID
	CUDAToolkitTarget       string
	CUDAToolkitTransitional []string
	TorchVariant            string
	Packages                []PackageDigest
}

// Profile returns one lockable profile definition.
func Profile(id ProfileID) (LockedProfile, error) {
	switch id {
	case ProfileAmpere:
		return LockedProfile{
			ID:                ProfileAmpere,
			GPUUUID:           AmpereGPUUUID,
			CUDAToolkitTarget: "13.0",
			TorchVariant:      "cu130",
			Packages: []PackageDigest{
				{Name: "torch", Digest: "sha256:cu130-compatible"},
			},
		}, nil
	case ProfilePascal:
		return LockedProfile{
			ID:                      ProfilePascal,
			GPUUUID:                 PascalGPUUUID,
			CUDAToolkitTarget:       "12.9",
			CUDAToolkitTransitional: []string{"12.4"},
			TorchVariant:            "cu124",
			Packages: []PackageDigest{
				{Name: "torch", Digest: "sha256:cu124-compatible"},
			},
		}, nil
	default:
		return LockedProfile{}, providerError(CodeInvalidProfile, "unknown environment profile %q", id)
	}
}

// AcceptsToolkitVersion reports whether an observed toolkit satisfies the profile.
func (profile LockedProfile) AcceptsToolkitVersion(version string) bool {
	if version == profile.CUDAToolkitTarget {
		return true
	}
	for _, allowed := range profile.CUDAToolkitTransitional {
		if version == allowed {
			return true
		}
	}
	return false
}

// ToolkitCandidates returns discovery order for one profile.
func (profile LockedProfile) ToolkitCandidates() []string {
	candidates := []string{profile.CUDAToolkitTarget}
	candidates = append(candidates, profile.CUDAToolkitTransitional...)
	return candidates
}

func (profile LockedProfile) artifactRequirements(toolkit string) gpu.ArtifactRequirements {
	return gpu.ArtifactRequirements{CUDAToolkit: toolkit, CompiledKernels: true}
}

func validateProfile(profile LockedProfile) error {
	if err := profile.GPUUUID.Validate(); err != nil {
		return providerError(CodeInvalidProfile, "profile GPU UUID: %v", err)
	}
	if profile.CUDAToolkitTarget == "" {
		return providerError(CodeInvalidProfile, "profile %q missing CUDA toolkit target", profile.ID)
	}
	return nil
}

func findDevice(devices []gpu.Device, uuid gpu.UUID) (gpu.Device, error) {
	for _, device := range devices {
		if device.UUID == uuid {
			return device, nil
		}
	}
	return gpu.Device{}, providerError(CodeGPUNotFound, "GPU %s not found in inventory", uuid)
}

func runtimeIdentity(profile LockedProfile) RuntimeIdentity {
	return RuntimeIdentity{
		PythonInterpreter: pythonInterpreterPlaceholder,
		TorchVariant:      profile.TorchVariant,
	}
}

func clonePackages(packages []PackageDigest) []PackageDigest {
	return append([]PackageDigest(nil), packages...)
}

func ensureProfile(id ProfileID) (LockedProfile, error) {
	profile, err := Profile(id)
	if err != nil {
		return LockedProfile{}, err
	}
	if err := validateProfile(profile); err != nil {
		return LockedProfile{}, err
	}
	return profile, nil
}

func validateRequest(request Request) error {
	if request.Version != ProviderVersion {
		return providerError(CodeInvalidRequest, "unsupported environment request version %q", request.Version)
	}
	if request.Profile == "" {
		return providerError(CodeInvalidRequest, "profile is required")
	}
	if _, err := ensureProfile(request.Profile); err != nil {
		return err
	}
	return nil
}
