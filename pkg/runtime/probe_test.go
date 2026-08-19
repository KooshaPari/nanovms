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
