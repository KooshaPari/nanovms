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
	// Executable reports that a configured command was found on PATH. It is
	// intentionally separate from Available: an installed command can still
	// fail its runtime health check.
	Executable bool
	// State distinguishes a missing/unusable runtime from an executable that
	// was found but could not be proven ready.
	State   AvailabilityState
	Reason  string
	Version string
}

// AvailabilityState is the local observation state for a runtime command.
// These values are diagnostic only and must not be persisted as provider
// metadata.
type AvailabilityState string

const (
	AvailabilityUnavailable    AvailabilityState = "unavailable"
	AvailabilityExecutableOnly AvailabilityState = "executable-only"
	AvailabilityReady          AvailabilityState = "ready"
)

// Ready reports whether the observation passed its configured health check.
func (a Availability) Ready() bool { return a.State == AvailabilityReady }

// Probe discovers local runtime availability. Implementations must not inspect
// cloud/provider state or credentials.
type Probe interface {
	Probe(context.Context, BackendID) Availability
}

// ProbeFunc adapts a function into a Probe.
type ProbeFunc func(context.Context, BackendID) Availability

// Probe implements Probe.
func (f ProbeFunc) Probe(ctx context.Context, backend BackendID) Availability { return f(ctx, backend) }

// CommandSpec describes the local command used to identify and health-check a
// runtime. It contains no provider or deployment metadata.
type CommandSpec struct {
	// Command is the preferred executable name or path.
	Command string
	// Candidates are optional executable aliases tried after Command.
	Candidates []string
	// VersionArgs identifies the executable and obtains its version.
	VersionArgs []string
	// HealthArgs performs a bounded, non-mutating runtime health check. An empty
	// list intentionally means executable-only probing for compatibility.
	HealthArgs []string
}

// BinaryProbe checks local runtime commands using explicit command specs.
type BinaryProbe struct {
	Commands   map[BackendID]string
	Candidates CandidateCommands
	Specs      map[BackendID]CommandSpec
	// Args and ReadinessArgs are compatibility maps for callers that predate
	// CommandSpec. CommandSpec values take precedence unless a compatibility
	// value is supplied for that backend.
	Args          map[BackendID][]string
	ReadinessArgs map[BackendID][]string
	// ArgRunner is the argument-aware execution hook used by tests and
	// embedders. A nil hook executes the real command with exec.CommandContext.
	ArgRunner func(context.Context, string, []string) ([]byte, error)
	Runner    Runner
	// Timeout bounds the complete version and health sequence. A zero value
	// uses the default two-second limit.
	Timeout time.Duration
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
func (p BinaryProbe) commandSpec(backend BackendID) CommandSpec {
	spec := p.Specs[backend]
	if command := p.Commands[backend]; command != "" {
		spec.Command = command
		spec.Candidates = nil
	} else if candidates := p.Candidates[backend]; len(candidates) != 0 {
		spec.Command = ""
		spec.Candidates = append([]string(nil), candidates...)
	}
	if args := p.Args[backend]; len(args) != 0 {
		spec.VersionArgs = append([]string(nil), args...)
	}
	if args := p.ReadinessArgs[backend]; len(args) != 0 {
		spec.HealthArgs = append([]string(nil), args...)
	}
	return spec
}

func (p BinaryProbe) candidates(backend BackendID) []string {
	spec := p.commandSpec(backend)
	commands := make([]string, 0, 1+len(spec.Candidates))
	if spec.Command != "" {
		commands = append(commands, spec.Command)
	}
	for _, candidate := range spec.Candidates {
		if candidate == "" {
			continue
		}
		seen := false
		for _, command := range commands {
			if command == candidate {
				seen = true
				break
			}
		}
		if !seen {
			commands = append(commands, candidate)
		}
	}
	return commands
}

// DefaultBinaryProbe returns local command and health mappings for supported
// container backends. WSLc is preferred, with Windows executable aliases
// retained for installations that ship an .exe name.
func DefaultBinaryProbe() BinaryProbe {
	const formatArg = "--format"

	return BinaryProbe{
		Commands: map[BackendID]string{
			BackendPodman:          "podman",
			BackendAppleContainers: "container",
		},
		Candidates: CandidateCommands{
			BackendWSLContainers: {"wslc", "wslc.exe", "container.exe"},
		},
		Specs: map[BackendID]CommandSpec{
			BackendPodman: {
				Command:     "podman",
				VersionArgs: []string{"version", formatArg, "{{.Version}}"},
				HealthArgs:  []string{"info"},
			},
			BackendAppleContainers: {
				Command:     "container",
				VersionArgs: []string{"system", "version", formatArg, "json"},
				HealthArgs:  []string{"system", "status", formatArg, "json"},
			},
			BackendWSLContainers: {
				Command:     "wslc",
				Candidates:  []string{"wslc.exe", "container.exe"},
				VersionArgs: []string{"version"},
				HealthArgs:  []string{"container", "list", "--all", "--quiet"},
			},
		},
		Args: map[BackendID][]string{
			BackendPodman:          {"version", formatArg, "{{.Version}}"},
			BackendAppleContainers: {"system", "version", formatArg, "json"},
			BackendWSLContainers:   {"version"},
		},
		ReadinessArgs: map[BackendID][]string{
			BackendPodman:          {"info"},
			BackendAppleContainers: {"system", "status", formatArg, "json"},
			BackendWSLContainers:   {"container", "list", "--all", "--quiet"},
		},
	}
}

// Probe checks PATH and runs the configured version and health commands with a
// bounded context. It never creates, starts, or mutates a runtime workload.
func (p BinaryProbe) Probe(ctx context.Context, backend BackendID) Availability {
	spec := p.commandSpec(backend)
	commands := p.candidates(backend)
	if len(commands) == 0 {
		return Availability{Backend: backend, State: AvailabilityUnavailable, Reason: "no local probe configured"}
	}
	var path string
	for _, command := range commands {
		if found, err := exec.LookPath(command); err == nil {
			path = found
			break
		}
	}
	if path == "" {
		return Availability{Backend: backend, State: AvailabilityUnavailable, Reason: "executable unavailable"}
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	versionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	versionArgs := spec.VersionArgs
	if len(versionArgs) == 0 {
		versionArgs = []string{"--version"}
	}
	out, err := p.run(versionCtx, path, versionArgs)
	if err != nil {
		if versionCtx.Err() != nil {
			return Availability{Backend: backend, Executable: true, State: AvailabilityUnavailable, Reason: "probe timed out", Version: "unknown"}
		}
		return Availability{Backend: backend, Available: true, Executable: true, State: AvailabilityExecutableOnly, Reason: "executable found", Version: "unknown"}
	}
	version := strings.TrimSpace(string(out))
	if len(spec.HealthArgs) == 0 {
		return Availability{Backend: backend, Available: true, Executable: true, State: AvailabilityExecutableOnly, Reason: "executable found", Version: version}
	}
	if _, err := p.run(versionCtx, path, spec.HealthArgs); err != nil {
		if versionCtx.Err() != nil {
			return Availability{Backend: backend, Executable: true, State: AvailabilityExecutableOnly, Reason: "health probe timed out", Version: version}
		}
		return Availability{Backend: backend, Executable: true, State: AvailabilityExecutableOnly, Reason: "runtime health check failed", Version: version}
	}
	return Availability{Backend: backend, Available: true, Executable: true, State: AvailabilityReady, Reason: "runtime ready", Version: version}
}

func (p BinaryProbe) run(ctx context.Context, path string, args []string) ([]byte, error) {
	if p.ArgRunner != nil {
		return p.ArgRunner(ctx, path, args)
	}
	if p.Runner != nil {
		return p.Runner(ctx, path)
	}
	return exec.CommandContext(ctx, path, args...).Output()
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
