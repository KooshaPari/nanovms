package runtime

import "testing"

func TestBackendCompatibilityMatrix(t *testing.T) {
	for _, backend := range []Backend{BackendPodman, BackendAppleContainers, BackendWSLContainers} {
		if !backend.Supports(PlanTargetDocker) || backend.Supports(PlanTargetNanoVMS) {
			t.Fatalf("%s compatibility mismatch", backend)
		}
	}
	if !BackendNanoVMS.Supports(PlanTargetNanoVMS) || BackendNanoVMS.Supports(PlanTargetDocker) {
		t.Fatal("nanovms compatibility mismatch")
	}
}

func TestParseBackendRejectsUnknownValues(t *testing.T) {
	if _, err := ParseBackend("aws"); err == nil {
		t.Fatal("expected unknown backend to be rejected")
	}
}
