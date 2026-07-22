// SPDX-License-Identifier: MIT OR Apache-2.0
// Package gpu provides GPU inventory, compatibility, and reservation contracts.
package gpu

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var nvidiaUUIDPattern = regexp.MustCompile(`^GPU-([0-9A-Fa-f]{8})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{4})-([0-9A-Fa-f]{12})$`)

// UUID is a canonical NVIDIA physical-GPU UUID. PCI addresses and observed
// indices are deliberately not accepted as identity.
type UUID string

// ParseUUID validates and canonicalizes an NVIDIA physical-GPU UUID.
func ParseUUID(value string) (UUID, error) {
	match := nvidiaUUIDPattern.FindStringSubmatch(value)
	if match == nil {
		return "", fmt.Errorf("invalid NVIDIA GPU UUID %q", value)
	}
	return UUID("GPU-" + strings.ToLower(strings.Join(match[1:], "-"))), nil
}

// Validate verifies that u is already in canonical form.
func (u UUID) Validate() error {
	canonical, err := ParseUUID(string(u))
	if err != nil {
		return err
	}
	if canonical != u {
		return fmt.Errorf("NVIDIA GPU UUID %q is not canonical (want %q)", u, canonical)
	}
	return nil
}

// InventoryScope identifies the namespace in which an index was observed.
type InventoryScope string

const (
	ScopeWindowsHost InventoryScope = "windows-host"
	ScopeWSLDistro   InventoryScope = "wsl-distro"
	ScopeRuntime     InventoryScope = "runtime"
)

// Observation records a scope-local index. Index is never an identity.
type Observation struct {
	Scope   InventoryScope `json:"scope"`
	ScopeID string         `json:"scope_id,omitempty"`
	Index   int            `json:"observed_index"`
}

// Device is reconciled physical-GPU inventory keyed only by UUID.
type Device struct {
	UUID              UUID          `json:"uuid"`
	Name              string        `json:"name"`
	Architecture      string        `json:"architecture,omitempty"`
	ComputeCapability string        `json:"compute_capability,omitempty"`
	DriverVersion     string        `json:"driver_version,omitempty"`
	DriverCUDACeiling string        `json:"driver_cuda_ceiling,omitempty"`
	Observations      []Observation `json:"observations"`
}

// ScopedInventory is one adapter's inventory result.
type ScopedInventory struct {
	Scope             InventoryScope `json:"scope"`
	ScopeID           string         `json:"scope_id,omitempty"`
	DriverVersion     string         `json:"driver_version,omitempty"`
	DriverCUDACeiling string         `json:"driver_cuda_ceiling,omitempty"`
	Devices           []Device       `json:"devices"`
}

// Reconcile merges inventories solely by canonical UUID. Missing devices in a
// narrower scope are retained as subset visibility, not inferred by index.
func Reconcile(inventories ...ScopedInventory) ([]Device, error) {
	merged := make(map[UUID]Device)
	for _, inventory := range inventories {
		if inventory.Scope != ScopeWindowsHost && inventory.Scope != ScopeWSLDistro && inventory.Scope != ScopeRuntime {
			return nil, fmt.Errorf("invalid inventory scope %q", inventory.Scope)
		}
		seen := make(map[UUID]struct{}, len(inventory.Devices))
		for _, candidate := range inventory.Devices {
			if err := candidate.UUID.Validate(); err != nil {
				return nil, err
			}
			if _, duplicate := seen[candidate.UUID]; duplicate {
				return nil, fmt.Errorf("duplicate GPU UUID %q in %s inventory", candidate.UUID, inventory.Scope)
			}
			seen[candidate.UUID] = struct{}{}
			if len(candidate.Observations) != 1 {
				return nil, fmt.Errorf("GPU %q in scoped inventory must have exactly one observation", candidate.UUID)
			}
			observation := candidate.Observations[0]
			if observation.Scope != inventory.Scope || observation.ScopeID != inventory.ScopeID || observation.Index < 0 {
				return nil, fmt.Errorf("GPU %q has invalid scope observation", candidate.UUID)
			}
			current, exists := merged[candidate.UUID]
			if !exists {
				current = candidate
				current.Observations = nil
			} else if current.Name != candidate.Name || incompatibleNonEmpty(current.ComputeCapability, candidate.ComputeCapability) || incompatibleNonEmpty(current.Architecture, candidate.Architecture) {
				return nil, fmt.Errorf("conflicting inventory facts for GPU %q", candidate.UUID)
			}
			fillDeviceFacts(&current, candidate, inventory)
			current.Observations = append(current.Observations, observation)
			merged[candidate.UUID] = current
		}
	}

	result := make([]Device, 0, len(merged))
	for _, device := range merged {
		sort.Slice(device.Observations, func(i, j int) bool {
			if device.Observations[i].Scope == device.Observations[j].Scope {
				return device.Observations[i].ScopeID < device.Observations[j].ScopeID
			}
			return device.Observations[i].Scope < device.Observations[j].Scope
		})
		result = append(result, device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UUID < result[j].UUID })
	return result, nil
}

func incompatibleNonEmpty(left, right string) bool {
	return left != "" && right != "" && !strings.EqualFold(left, right)
}

func fillDeviceFacts(target *Device, source Device, inventory ScopedInventory) {
	if target.Architecture == "" {
		target.Architecture = source.Architecture
	}
	if target.ComputeCapability == "" {
		target.ComputeCapability = source.ComputeCapability
	}
	if target.DriverVersion == "" {
		target.DriverVersion = firstNonEmpty(source.DriverVersion, inventory.DriverVersion)
	}
	if target.DriverCUDACeiling == "" {
		target.DriverCUDACeiling = firstNonEmpty(source.DriverCUDACeiling, inventory.DriverCUDACeiling)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// ArtifactRequirements declares the CUDA artifact being dispatched. The
// driver ceiling is intentionally not treated as installed-toolkit evidence.
type ArtifactRequirements struct {
	CUDAToolkit     string `json:"cuda_toolkit,omitempty"`
	CompiledKernels bool   `json:"compiled_kernels,omitempty"`
}

// ValidateCompatibility checks declared artifact requirements against a GPU.
// Architecture and driver CUDA ceiling gates apply to both precompiled and
// compiler-required artifacts. The ceiling is still not proof of installed nvcc.
func ValidateCompatibility(device Device, artifact ArtifactRequirements) error {
	if strings.TrimSpace(artifact.CUDAToolkit) == "" {
		if artifact.CompiledKernels {
			return fmt.Errorf("compiled-kernel artifact must declare CUDA toolkit")
		}
		return nil
	}
	toolkitMajor, _, err := parseMajorMinor(artifact.CUDAToolkit)
	if err != nil {
		if artifact.CompiledKernels {
			return fmt.Errorf("compiled-kernel artifact must declare CUDA toolkit: %w", err)
		}
		return fmt.Errorf("artifact CUDA toolkit is invalid: %w", err)
	}
	architecture := strings.ToLower(device.Architecture)
	computeMajor, _, computeErr := parseMajorMinor(device.ComputeCapability)
	isPascal := strings.Contains(architecture, "pascal") || computeErr == nil && computeMajor == 6
	isAmpere := strings.Contains(architecture, "ampere") || computeErr == nil && computeMajor == 8
	if toolkitMajor >= 13 && isPascal {
		return fmt.Errorf("CUDA %s compiled kernels are incompatible with Pascal GPU %s", artifact.CUDAToolkit, device.UUID)
	}
	if toolkitMajor >= 13 && !isAmpere && architecture == "" && computeErr != nil {
		return fmt.Errorf("cannot prove CUDA %s compatibility for GPU %s without architecture or compute capability", artifact.CUDAToolkit, device.UUID)
	}
	if device.DriverCUDACeiling != "" {
		ceilingMajor, ceilingMinor, ceilingErr := parseMajorMinor(device.DriverCUDACeiling)
		toolkitMajor, toolkitMinor, _ := parseMajorMinor(artifact.CUDAToolkit)
		if ceilingErr == nil && (ceilingMajor < toolkitMajor || ceilingMajor == toolkitMajor && ceilingMinor < toolkitMinor) {
			return fmt.Errorf("driver CUDA ceiling %s is below artifact toolkit %s for GPU %s", device.DriverCUDACeiling, artifact.CUDAToolkit, device.UUID)
		}
	}
	return nil
}

func parseMajorMinor(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "cuda ")), ".")
	if len(parts) < 1 || parts[0] == "" {
		return 0, 0, fmt.Errorf("invalid version %q", value)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid version %q", value)
	}
	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid version %q", value)
		}
	}
	return major, minor, nil
}

const ResourceManifestVersion = "nanovms.io/resources/v1"

// ResourceManifest is the versioned companion to a composition request.
type ResourceManifest struct {
	Version  string               `json:"version"`
	GPUs     []Device             `json:"gpus,omitempty"`
	Artifact ArtifactRequirements `json:"artifact,omitempty"`
}

// CanonicalJSON validates and deterministically serializes the manifest.
func (manifest ResourceManifest) CanonicalJSON() ([]byte, error) {
	if manifest.Version != ResourceManifestVersion {
		return nil, fmt.Errorf("unsupported resource manifest version %q", manifest.Version)
	}
	copyManifest := manifest
	copyManifest.GPUs = append([]Device(nil), manifest.GPUs...)
	for i := range copyManifest.GPUs {
		copyManifest.GPUs[i].Observations = append([]Observation(nil), copyManifest.GPUs[i].Observations...)
	}
	sort.Slice(copyManifest.GPUs, func(i, j int) bool { return copyManifest.GPUs[i].UUID < copyManifest.GPUs[j].UUID })
	seen := make(map[UUID]struct{}, len(copyManifest.GPUs))
	for i := range copyManifest.GPUs {
		if err := copyManifest.GPUs[i].UUID.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[copyManifest.GPUs[i].UUID]; duplicate {
			return nil, fmt.Errorf("duplicate GPU UUID %q in resource manifest", copyManifest.GPUs[i].UUID)
		}
		seen[copyManifest.GPUs[i].UUID] = struct{}{}
		sort.Slice(copyManifest.GPUs[i].Observations, func(a, b int) bool {
			left, right := copyManifest.GPUs[i].Observations[a], copyManifest.GPUs[i].Observations[b]
			if left.Scope == right.Scope {
				if left.ScopeID == right.ScopeID {
					return left.Index < right.Index
				}
				return left.ScopeID < right.ScopeID
			}
			return left.Scope < right.Scope
		})
	}
	return json.Marshal(copyManifest)
}
