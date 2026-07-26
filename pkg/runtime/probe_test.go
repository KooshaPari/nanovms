package runtime

import (
	"context"
	"testing"
)

func TestBinaryProbeDistinguishesUnconfiguredAndUnavailable(t *testing.T) {
	probe := BinaryProbe{Commands: map[BackendID]string{
		BackendPodman: "definitely-not-installed-nanovms-test",
	}}
	if got := probe.Probe(context.Background(), BackendAppleContainers); got.State != AvailabilityUnconfigured {
		t.Fatalf("unconfigured state = %q, want %q", got.State, AvailabilityUnconfigured)
	}
	if got := probe.Probe(context.Background(), BackendPodman); got.State != AvailabilityUnavailable || got.Available {
		t.Fatalf("missing executable observation = %#v", got)
	}
}

func TestCommandProbeReportsReadinessSeparatelyFromExecutable(t *testing.T) {
	probe := CommandProbe{
		Commands: map[BackendID]string{BackendPodman: "go"},
		Args: map[BackendID][]string{
			BackendPodman: {"version"},
		},
	}
	if got := probe.Probe(context.Background(), BackendPodman); got.State != AvailabilityAvailable || !got.Available {
		t.Fatalf("successful readiness observation = %#v", got)
	}
	probe.Args[BackendPodman] = []string{"this-subcommand-does-not-exist"}
	if got := probe.Probe(context.Background(), BackendPodman); got.State != AvailabilityUnavailable || got.Available {
		t.Fatalf("failed readiness observation = %#v", got)
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
