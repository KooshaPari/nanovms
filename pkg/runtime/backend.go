package runtime

import "fmt"

// Backend identifies an execution substrate. Backends are runtime adapters and
// do not own cloud provider state.
type Backend string

const (
	// BackendNanoVMS consumes NanoVMS execution plans.
	BackendNanoVMS Backend = "nanovms"
	// BackendPodman consumes Docker-compatible OCI plans.
	BackendPodman Backend = "podman"
	// BackendAppleContainers consumes Docker-compatible OCI plans.
	BackendAppleContainers Backend = "apple-containers"
	// BackendWSLContainers is the first-party WSL containers extension.
	BackendWSLContainers Backend = "wsl-containers"
)

// PlanTarget identifies the renderer format supplied by PhenoCompose.
type PlanTarget string

const (
	// PlanTargetDocker is a Docker Compose / OCI plan.
	PlanTargetDocker PlanTarget = "docker"
	// PlanTargetNanoVMS is a NanoVMS execution plan.
	PlanTargetNanoVMS PlanTarget = "nanovms"
)

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

// Supports reports whether the backend can consume a plan target.
func (b Backend) Supports(target PlanTarget) bool {
	if b == BackendNanoVMS {
		return target == PlanTargetNanoVMS
	}
	return (b == BackendPodman || b == BackendAppleContainers || b == BackendWSLContainers) && target == PlanTargetDocker
}
