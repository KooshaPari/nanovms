package runtime

import (
	"context"
	"testing"
	"time"
)

func TestBinaryProbeReportsUnconfiguredAndMissingCommands(t *testing.T) {
	probe := BinaryProbe{Commands: map[BackendID]string{BackendPodman: "definitely-not-installed-nvms"}}
	if got := probe.Probe(context.Background(), BackendAppleContainers); got.Reason != "no local probe configured" {
		t.Fatalf("unexpected unconfigured result: %#v", got)
	}
	if got := probe.Probe(context.Background(), BackendPodman); got.Reason != "executable unavailable" {
		t.Fatalf("unexpected missing executable result: %#v", got)
	}
	if got := probe.Probe(context.Background(), BackendPodman); got.State != AvailabilityUnavailable || got.Executable {
		t.Fatalf("missing executable was not reported unavailable: %#v", got)
	}
}

func TestBinaryProbeReportsVersionTimeout(t *testing.T) {
	probe := BinaryProbe{
		Commands: map[BackendID]string{BackendPodman: "go"},
		Timeout:  50 * time.Millisecond,
		Runner: func(ctx context.Context, _ string) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	started := time.Now()
	got := probe.Probe(context.Background(), BackendPodman)
	if got.Available || got.State != AvailabilityUnavailable || got.Reason != "probe timed out" {
		t.Fatalf("unexpected timeout result: %#v", got)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("probe exceeded bounded timeout: %s", elapsed)
	}
}

func TestDefaultBinaryProbeIncludesCurrentAndLegacyWSLCNames(t *testing.T) {
	probe := DefaultBinaryProbe()
	got := probe.Candidates[BackendWSLContainers]
	if len(got) != 3 || got[0] != "wslc" || got[1] != "wslc.exe" || got[2] != "container.exe" {
		t.Fatalf("unexpected WSLC candidates: %#v", got)
	}
	podman := probe.Specs[BackendPodman]
	if podman.Command != "podman" || !equalStrings(podman.VersionArgs, []string{"version", "--format", "{{.Version}}"}) || !equalStrings(podman.HealthArgs, []string{"info"}) {
		t.Fatalf("unexpected Podman command spec: %#v", podman)
	}
	apple := probe.Specs[BackendAppleContainers]
	if apple.Command != "container" || !equalStrings(apple.VersionArgs, []string{"system", "version", "--format", "json"}) || !equalStrings(apple.HealthArgs, []string{"system", "status", "--format", "json"}) {
		t.Fatalf("unexpected Apple Containers command spec: %#v", apple)
	}
	wsl := probe.Specs[BackendWSLContainers]
	if wsl.Command != "wslc" || !equalStrings(wsl.VersionArgs, []string{"version"}) || !equalStrings(wsl.HealthArgs, []string{"container", "list", "--all", "--quiet"}) {
		t.Fatalf("unexpected WSLc command spec: %#v", wsl)
	}
}

func TestBinaryProbeUsesExplicitVersionAndHealthArguments(t *testing.T) {
	var calls [][]string
	probe := BinaryProbe{
		Specs: map[BackendID]CommandSpec{
			BackendPodman: {
				Command:     "go",
				VersionArgs: []string{"version"},
				HealthArgs:  []string{"env"},
			},
		},
		ArgRunner: func(_ context.Context, _ string, args []string) ([]byte, error) {
			calls = append(calls, append([]string(nil), args...))
			if len(calls) == 1 {
				return []byte("go1.25"), nil
			}
			return nil, nil
		},
	}
	got := probe.Probe(context.Background(), BackendPodman)
	if !got.Available || !got.Executable || !got.Ready() || got.State != AvailabilityReady || got.Version != "go1.25" {
		t.Fatalf("unexpected ready result: %#v", got)
	}
	want := [][]string{{"version"}, {"env"}}
	if !equalStringLists(calls, want) {
		t.Fatalf("commands = %#v, want %#v", calls, want)
	}
}

func TestBinaryProbeReportsHealthFailureAsExecutableOnly(t *testing.T) {
	probe := BinaryProbe{
		Specs: map[BackendID]CommandSpec{
			BackendPodman: {
				Command:     "go",
				VersionArgs: []string{"version"},
				HealthArgs:  []string{"env"},
			},
		},
		ArgRunner: func(_ context.Context, _ string, args []string) ([]byte, error) {
			if len(args) == 1 && args[0] == "version" {
				return []byte("go1.25"), nil
			}
			return nil, context.Canceled
		},
	}
	got := probe.Probe(context.Background(), BackendPodman)
	if got.Available || !got.Executable || got.State != AvailabilityExecutableOnly || got.Reason != "runtime health check failed" || got.Version != "go1.25" {
		t.Fatalf("unexpected health failure result: %#v", got)
	}
}

func TestBinaryProbeReportsExecutableOnlyWhenHealthIsNotConfigured(t *testing.T) {
	probe := BinaryProbe{
		Specs: map[BackendID]CommandSpec{
			BackendPodman: {Command: "go", VersionArgs: []string{"version"}},
		},
		ArgRunner: func(_ context.Context, _ string, _ []string) ([]byte, error) {
			return []byte("go1.25"), nil
		},
	}
	got := probe.Probe(context.Background(), BackendPodman)
	if !got.Available || !got.Executable || got.State != AvailabilityExecutableOnly {
		t.Fatalf("unexpected executable-only result: %#v", got)
	}
}

func TestSelectHonorsPreferenceAndNeverLeaksCloudState(t *testing.T) {
	probe := ProbeFunc(func(_ context.Context, backend BackendID) Availability {
		return Availability{Backend: backend, Available: backend == BackendPodman || backend == BackendWSLContainers, Reason: "fake"}
	})
	metadata, observed, err := Select(context.Background(), NewBackendRegistry(), probe, PlanTargetDocker, []BackendID{BackendWSLContainers})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ID != BackendWSLContainers || observed.Backend != BackendWSLContainers {
		t.Fatalf("unexpected selection: %#v %#v", metadata, observed)
	}
}

func TestDiscoverIsDeterministicAndReportsUnavailable(t *testing.T) {
	observed := Discover(context.Background(), NewBackendRegistry(), ProbeFunc(func(_ context.Context, backend BackendID) Availability {
		return Availability{Backend: backend, Available: backend == BackendNanoVMS}
	}))
	if len(observed) != 4 || observed[0].Backend != BackendAppleContainers || observed[0].Available {
		t.Fatalf("unexpected observations: %#v", observed)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func equalStringLists(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !equalStrings(got[i], want[i]) {
			return false
		}
	}
	return true
}
