package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Availability describes an observed local backend without becoming persisted
// deployment metadata.
type Availability struct {
	Backend   BackendID
	Available bool
	// State is a machine-readable result for audit and scheduler decisions.
	// Available is retained for compatibility with existing callers.
	State   AvailabilityState
	Reason  string
	Version string
}

// AvailabilityState distinguishes an unconfigured adapter from an installed
// but unusable engine. These observations are local-only and must not be
// persisted as provider deployment state.
type AvailabilityState string

const (
	AvailabilityUnknown      AvailabilityState = "unknown"
	AvailabilityAvailable    AvailabilityState = "available"
	AvailabilityUnavailable  AvailabilityState = "unavailable"
	AvailabilityUnconfigured AvailabilityState = "unconfigured"
)

// Probe discovers local runtime availability. Implementations must not inspect
// cloud/provider state or credentials.
type Probe interface {
	Probe(context.Context, BackendID) Availability
}

// ProbeFunc adapts a function into a Probe.
type ProbeFunc func(context.Context, BackendID) Availability

// Probe implements Probe.
func (f ProbeFunc) Probe(ctx context.Context, backend BackendID) Availability { return f(ctx, backend) }

// BinaryProbe checks local executable availability only.
type BinaryProbe struct{ Commands map[BackendID]string }

// Probe checks PATH and, when present, obtains a bounded version string.
func (p BinaryProbe) Probe(ctx context.Context, backend BackendID) Availability {
	command := p.Commands[backend]
	if command == "" {
		return Availability{Backend: backend, State: AvailabilityUnconfigured, Reason: "no local probe configured"}
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return Availability{Backend: backend, State: AvailabilityUnavailable, Reason: "executable unavailable"}
	}
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(versionCtx, path, "--version").Output()
	if err != nil {
		return Availability{Backend: backend, Available: true, State: AvailabilityAvailable, Reason: "executable found", Version: "unknown"}
	}
	return Availability{Backend: backend, Available: true, State: AvailabilityAvailable, Reason: "executable found", Version: strings.TrimSpace(string(out))}
}

// CommandProbe validates an engine with a read-only command (for example
// `docker info` or `podman info`). It deliberately does not create resources.
// Commands and Args are keyed by canonical BackendID values.
type CommandProbe struct {
	Commands map[BackendID]string
	Args     map[BackendID][]string
}

// Probe executes the configured read-only readiness command with a bounded
// timeout. A non-zero exit means the executable exists but the engine is not
// ready, which is different from an unconfigured adapter.
func (p CommandProbe) Probe(ctx context.Context, backend BackendID) Availability {
	command := p.Commands[backend]
	if command == "" {
		return Availability{Backend: backend, State: AvailabilityUnconfigured, Reason: "no readiness probe configured"}
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return Availability{Backend: backend, State: AvailabilityUnavailable, Reason: "executable unavailable"}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := exec.CommandContext(probeCtx, path, p.Args[backend]...).Run(); err != nil {
		return Availability{Backend: backend, State: AvailabilityUnavailable, Reason: "readiness probe failed"}
	}
	return Availability{Backend: backend, Available: true, State: AvailabilityAvailable, Reason: "readiness probe passed"}
}

// Discover probes all registered backends in deterministic ID order.
func Discover(ctx context.Context, registry *BackendRegistry, probe Probe) []Availability {
	if registry == nil || probe == nil {
		return nil
	}
	backends := registry.All()
	result := make([]Availability, 0, len(backends))
	for _, metadata := range backends {
		result = append(result, probe.Probe(ctx, metadata.ID))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Backend < result[j].Backend })
	return result
}

// Select chooses an available backend for a target, honoring preference first
// and canonical ID order second.
func Select(ctx context.Context, registry *BackendRegistry, probe Probe, target PlanTarget, preferred []BackendID) (BackendMetadata, Availability, error) {
	if registry == nil || probe == nil {
		return BackendMetadata{}, Availability{}, fmt.Errorf("runtime registry and probe are required")
	}
	available := map[BackendID]Availability{}
	for _, observation := range Discover(ctx, registry, probe) {
		available[observation.Backend] = observation
	}
	try := append([]BackendID{}, preferred...)
	for _, metadata := range registry.All() {
		try = append(try, metadata.ID)
	}
	seen := map[BackendID]bool{}
	for _, id := range try {
		if seen[id] {
			continue
		}
		seen[id] = true
		metadata, err := registry.Resolve(id)
		if err != nil || !id.Supports(target) {
			continue
		}
		observation := available[id]
		if observation.Available {
			return metadata, observation, nil
		}
	}
	return BackendMetadata{}, Availability{}, fmt.Errorf("no available backend supports target %q", target)
}
