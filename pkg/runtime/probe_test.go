package runtime

import (
	"context"
	"testing"
)

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
