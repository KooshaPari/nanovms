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
	Reason    string
	Version   string
}

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
		return Availability{Backend: backend, Reason: "no local probe configured"}
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return Availability{Backend: backend, Reason: "executable unavailable"}
	}
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(versionCtx, path, "--version").Output()
	if err != nil {
		return Availability{Backend: backend, Available: true, Reason: "executable found", Version: "unknown"}
	}
	return Availability{Backend: backend, Available: true, Reason: "executable found", Version: strings.TrimSpace(string(out))}
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
