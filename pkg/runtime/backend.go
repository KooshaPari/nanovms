// SPDX-License-Identifier: MIT OR Apache-2.0
package runtime

import (
	"fmt"
	"sort"
)

// BackendID identifies an execution backend. Backend records describe local
// execution capabilities only; they do not contain cloud resources or state.
type BackendID string

// Backend is retained as the short name used by existing callers.
type Backend = BackendID

// PlanTarget identifies a renderer format supplied by PhenoCompose.
type PlanTarget string

const (
	// PlanTargetDocker is a Docker Compose / OCI plan.
	PlanTargetDocker PlanTarget = "docker"
	// PlanTargetNanoVMS is a NanoVMS execution plan.
	PlanTargetNanoVMS PlanTarget = "nanovms"
)

const (
	// BackendNanoVMS is NanoVMS' native VM/process abstraction.
	BackendNanoVMS BackendID = "nanovms"
	// BackendPodman is the Podman OCI container backend.
	BackendPodman BackendID = "podman"
	// BackendAppleContainers is Apple's native container backend.
	BackendAppleContainers BackendID = "apple-containers"
	// BackendWSLContainers is the maintained WSL container backend.
	BackendWSLContainers BackendID = "wsl-containers"
)

// BackendMetadata is the immutable capability metadata used when selecting a
// local execution backend for a composition.
type BackendMetadata struct {
	// ID is the canonical backend identifier.
	ID BackendID
	// Tier is the existing NanoVMS isolation tier used by the orchestration
	// engine when this backend is selected.
	Tier int
	// Lifecycle indicates that deploy/start lifecycle calls are supported.
	Lifecycle bool
}

// BackendRegistry provides deterministic lookup of supported local backends.
type BackendRegistry struct {
	backends map[BackendID]BackendMetadata
}

// NewBackendRegistry returns the built-in backend capability matrix.
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{backends: map[BackendID]BackendMetadata{
		BackendNanoVMS:         {ID: BackendNanoVMS, Tier: 3, Lifecycle: true},
		BackendPodman:          {ID: BackendPodman, Tier: 2, Lifecycle: true},
		BackendAppleContainers: {ID: BackendAppleContainers, Tier: 2, Lifecycle: true},
		BackendWSLContainers:   {ID: BackendWSLContainers, Tier: 2, Lifecycle: true},
	}}
}

// Resolve returns the capability metadata for id.
func (r *BackendRegistry) Resolve(id BackendID) (BackendMetadata, error) {
	if r == nil {
		return BackendMetadata{}, fmt.Errorf("backend registry is nil")
	}
	metadata, ok := r.backends[id]
	if !ok {
		return BackendMetadata{}, fmt.Errorf("unsupported execution backend %q", id)
	}
	return metadata, nil
}

// All returns capabilities sorted by canonical backend identifier.
func (r *BackendRegistry) All() []BackendMetadata {
	if r == nil {
		return nil
	}
	result := make([]BackendMetadata, 0, len(r.backends))
	for _, metadata := range r.backends {
		result = append(result, metadata)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// ParseBackend validates a configured backend identifier.
func ParseBackend(value string) (Backend, error) {
	b := Backend(value)
	switch b {
	case BackendNanoVMS, BackendPodman, BackendAppleContainers, BackendWSLContainers:
		return b, nil
	default:
		return "", fmt.Errorf("unsupported execution backend %q", value)
	}
}

// Supports reports whether the backend can consume a renderer target.
func (b BackendID) Supports(target PlanTarget) bool {
	if b == BackendNanoVMS {
		return target == PlanTargetNanoVMS
	}
	return (b == BackendPodman || b == BackendAppleContainers || b == BackendWSLContainers) && target == PlanTargetDocker
}
