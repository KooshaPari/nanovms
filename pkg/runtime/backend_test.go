// SPDX-License-Identifier: MIT OR Apache-2.0
package runtime

import "testing"

func TestBackendRegistryIsDeterministic(t *testing.T) {
	registry := NewBackendRegistry()
	all := registry.All()
	want := []BackendID{BackendAppleContainers, BackendNanoVMS, BackendPodman, BackendWSLContainers}
	if len(all) != len(want) {
		t.Fatalf("got %d backends, want %d", len(all), len(want))
	}
	for i, metadata := range all {
		if metadata.ID != want[i] {
			t.Errorf("backend %d = %q, want %q", i, metadata.ID, want[i])
		}
		if !metadata.Lifecycle || metadata.Tier < 1 {
			t.Errorf("backend %q has invalid lifecycle metadata: %+v", metadata.ID, metadata)
		}
	}
}

func TestBackendRegistryRejectsUnknownBackend(t *testing.T) {
	if _, err := NewBackendRegistry().Resolve("unknown"); err == nil {
		t.Fatal("expected unknown backend to be rejected")
	}
}

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
