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
}

func TestBinaryProbeReportsVersionTimeout(t *testing.T) {
	probe := BinaryProbe{
		Commands: map[BackendID]string{BackendPodman: "go"},
		Runner: func(ctx context.Context, _ string) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	started := time.Now()
	got := probe.Probe(context.Background(), BackendPodman)
	if got.Available || got.Reason != "probe timed out" {
		t.Fatalf("unexpected timeout result: %#v", got)
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("probe exceeded bounded timeout: %s", elapsed)
	}
}

func TestDefaultBinaryProbeIncludesCurrentAndLegacyWSLCNames(t *testing.T) {
	probe := DefaultBinaryProbe()
	got := probe.Candidates[BackendWSLContainers]
	if len(got) != 2 || got[0] != "container.exe" || got[1] != "wslc.exe" {
		t.Fatalf("unexpected WSLC candidates: %#v", got)
	}
	if got := probe.Args[BackendAppleContainers]; len(got) != 4 || got[0] != "system" || got[1] != "version" || got[2] != "--format" || got[3] != "json" {
		t.Fatalf("unexpected Apple Containers probe args: %#v", got)
	}
	if got := probe.Args[BackendWSLContainers]; len(got) != 1 || got[0] != "version" {
		t.Fatalf("unexpected WSL Containers probe args: %#v", got)
	}
	if got := probe.Platforms[BackendAppleContainers]; len(got) != 1 || got[0] != "darwin" {
		t.Fatalf("unexpected Apple Containers platforms: %#v", got)
	}
	if got := probe.Platforms[BackendWSLContainers]; len(got) != 1 || got[0] != "windows" {
		t.Fatalf("unexpected WSL Containers platforms: %#v", got)
	}
}

func TestBinaryProbeFailsClosedWhenVersionCommandFails(t *testing.T) {
	probe := BinaryProbe{
		Commands: map[BackendID]string{BackendPodman: "go"},
		ArgRunner: func(context.Context, string, []string) ([]byte, error) {
			return nil, context.Canceled
		},
	}
	got := probe.Probe(context.Background(), BackendPodman)
	if got.Available || got.Reason != "probe failed" {
		t.Fatalf("unexpected failed probe result: %#v", got)
	}
}

func TestBinaryProbeRequiresRuntimeReadiness(t *testing.T) {
	probe := BinaryProbe{
		Commands:      map[BackendID]string{BackendPodman: "go"},
		Args:          map[BackendID][]string{BackendPodman: {"--version"}},
		ReadinessArgs: map[BackendID][]string{BackendPodman: {"ps", "--all", "--noheading"}},
		ArgRunner: func(_ context.Context, _ string, args []string) ([]byte, error) {
			if len(args) == 1 && args[0] == "--version" {
				return []byte("podman 5.8.3"), nil
			}
			return nil, context.Canceled
		},
	}
	got := probe.Probe(context.Background(), BackendPodman)
	if got.Available || got.Reason != "runtime unavailable" || got.Version != "podman 5.8.3" {
		t.Fatalf("unexpected readiness result: %#v", got)
	}
}

func TestBinaryProbeUsesBackendSpecificArguments(t *testing.T) {
	var observed []string
	probe := BinaryProbe{
		Commands: map[BackendID]string{BackendAppleContainers: "go"},
		Args:     map[BackendID][]string{BackendAppleContainers: {"system", "version", "--format", "json"}},
		ArgRunner: func(_ context.Context, _ string, args []string) ([]byte, error) {
			observed = append([]string(nil), args...)
			return []byte("[{\"version\":\"1.0\"}]"), nil
		},
	}
	got := probe.Probe(context.Background(), BackendAppleContainers)
	if !got.Available || got.Version != `[{"version":"1.0"}]` {
		t.Fatalf("unexpected probe result: %#v", got)
	}
	want := []string{"system", "version", "--format", "json"}
	if len(observed) != len(want) {
		t.Fatalf("observed args = %#v, want %#v", observed, want)
	}
	for i := range want {
		if observed[i] != want[i] {
			t.Fatalf("observed args = %#v, want %#v", observed, want)
		}
	}
}

func TestSelectSkipsProbeOnlyPreference(t *testing.T) {
	registry := NewBackendRegistry()
	registry.backends[BackendWSLContainers] = BackendMetadata{ID: BackendWSLContainers, Tier: 2, Lifecycle: false}
	probe := ProbeFunc(func(_ context.Context, backend BackendID) Availability {
		return Availability{Backend: backend, Available: backend == BackendPodman || backend == BackendWSLContainers, Reason: "fake"}
	})
	metadata, observed, err := Select(context.Background(), registry, probe, PlanTargetDocker, []BackendID{BackendWSLContainers})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ID != BackendPodman || observed.Backend != BackendPodman {
		t.Fatalf("unexpected selection: %#v %#v", metadata, observed)
	}
}

func TestSelectRejectsProbeOnlyBackendsWhenNoLifecycleBackendIsAvailable(t *testing.T) {
	registry := NewBackendRegistry()
	registry.backends[BackendAppleContainers] = BackendMetadata{ID: BackendAppleContainers, Tier: 2, Lifecycle: false}
	registry.backends[BackendWSLContainers] = BackendMetadata{ID: BackendWSLContainers, Tier: 2, Lifecycle: false}
	probe := ProbeFunc(func(_ context.Context, backend BackendID) Availability {
		return Availability{Backend: backend, Available: backend == BackendAppleContainers || backend == BackendWSLContainers}
	})
	if _, _, err := Select(context.Background(), registry, probe, PlanTargetDocker, nil); err == nil {
		t.Fatal("expected probe-only backends to be rejected for deployment selection")
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
