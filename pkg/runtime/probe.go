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
type BinaryProbe struct {
	Commands   map[BackendID]string
	Candidates CandidateCommands
	Runner     Runner
}

// CandidateCommands optionally supplies fallbacks for a backend. The first
// executable found on PATH is used. This is useful for the WSL Containers CLI,
// whose shipped executable is currently named container.exe while older
// integrations used wslc.exe.
type CandidateCommands map[BackendID][]string

// Runner allows tests and embedders to provide a bounded version invocation.
// A nil Runner uses exec.CommandContext.
type Runner func(context.Context, string) ([]byte, error)

// Candidates and Runner are optional extensions to BinaryProbe. Commands is
// retained as the single-command compatibility path for existing callers.
// If both are supplied, Commands takes precedence for that backend.
func (p BinaryProbe) candidates(backend BackendID) []string {
	if command := p.Commands[backend]; command != "" {
		return []string{command}
	}
	if candidates := p.Candidates[backend]; len(candidates) != 0 {
		return candidates
	}
	return nil
}

// DefaultBinaryProbe returns local command mappings for supported container
// backends. The WSL Containers executable is tried under both its current
// container.exe name and the legacy wslc.exe name.
func DefaultBinaryProbe() BinaryProbe {
	return BinaryProbe{Commands: map[BackendID]string{
		BackendPodman:          "podman",
		BackendAppleContainers: "container",
	}, Candidates: CandidateCommands{
		BackendWSLContainers: {"container.exe", "wslc.exe"},
	}}
}

// Probe checks PATH and, when present, obtains a bounded version string.
func (p BinaryProbe) Probe(ctx context.Context, backend BackendID) Availability {
	commands := p.candidates(backend)
	if len(commands) == 0 {
		return Availability{Backend: backend, Reason: "no local probe configured"}
	}
	var path string
	for _, command := range commands {
		if found, err := exec.LookPath(command); err == nil {
			path = found
			break
		}
	}
	if path == "" {
		return Availability{Backend: backend, Reason: "executable unavailable"}
	}
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var out []byte
	var err error
	if p.Runner != nil {
		out, err = p.Runner(versionCtx, path)
	} else {
		out, err = exec.CommandContext(versionCtx, path, "--version").Output()
	}
	if err != nil {
		if versionCtx.Err() != nil {
			return Availability{Backend: backend, Reason: "probe timed out", Version: "unknown"}
		}
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
