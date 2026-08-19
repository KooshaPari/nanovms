// SPDX-License-Identifier: MIT OR Apache-2.0
package runtime

import (
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// --- Registry Tests ---

func TestRegistryNew(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	all := r.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(all))
	}
}

func TestRegistryGetTier1(t *testing.T) {
	r := NewRegistry()
	rt, err := r.Get(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "wasm" {
		t.Errorf("expected wasm, got %s", rt.Name())
	}
	if rt.Tier() != 1 {
		t.Errorf("expected tier 1, got %d", rt.Tier())
	}
}

func TestRegistryGetTier2(t *testing.T) {
	r := NewRegistry()
	rt, err := r.Get(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "gvisor" {
		t.Errorf("expected gvisor, got %s", rt.Name())
	}
}

func TestRegistryGetTier3(t *testing.T) {
	r := NewRegistry()
	rt, err := r.Get(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "firecracker" {
		t.Errorf("expected firecracker, got %s", rt.Name())
	}
}

func TestRegistryGetInvalid(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get(99)
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
}

func TestRegistryRegisterCustom(t *testing.T) {
	r := NewRegistry()
	custom := &mockRuntime{name: "custom", tier: 42}
	r.Register(42, custom)
	rt, err := r.Get(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "custom" {
		t.Errorf("expected custom, got %s", rt.Name())
	}
}

func TestRegistryRegisterNilMap(t *testing.T) {
	r := &Registry{}
	custom := &mockRuntime{name: "new", tier: 1}
	r.Register(1, custom)
	if r.runtimes == nil {
		t.Fatal("runtimes map should be initialized")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
	names := map[string]bool{}
	for _, rt := range all {
		names[rt.Name()] = true
	}
	for _, expected := range []string{"wasm", "gvisor", "firecracker"} {
		if !names[expected] {
			t.Errorf("missing runtime: %s", expected)
		}
	}
}

// --- BackendRegistry Tests ---

func TestBackendRegistryAll(t *testing.T) {
	r := NewBackendRegistry()
	all := r.All()
	if len(all) != 4 {
		t.Fatalf("expected 4 backends, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Errorf("backends not sorted: %s >= %s", all[i-1].ID, all[i].ID)
		}
	}
}

func TestBackendRegistryResolve(t *testing.T) {
	r := NewBackendRegistry()
	meta, err := r.Resolve(BackendPodman)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Tier != 2 {
		t.Errorf("expected tier 2, got %d", meta.Tier)
	}
	if !meta.Lifecycle {
		t.Error("expected lifecycle support")
	}
}

func TestBackendRegistryResolveInvalid(t *testing.T) {
	r := NewBackendRegistry()
	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for unsupported backend")
	}
}

func TestBackendRegistryResolveNil(t *testing.T) {
	var r *BackendRegistry
	_, err := r.Resolve(BackendNanoVMS)
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestBackendRegistryAllNil(t *testing.T) {
	var r *BackendRegistry
	if all := r.All(); all != nil {
		t.Errorf("expected nil, got %d", len(all))
	}
}

func TestParseBackendValid(t *testing.T) {
	for _, valid := range []Backend{BackendNanoVMS, BackendPodman, BackendAppleContainers, BackendWSLContainers} {
		b, err := ParseBackend(string(valid))
		if err != nil {
			t.Errorf("unexpected error for %s: %v", valid, err)
		}
		if b != valid {
			t.Errorf("expected %s, got %s", valid, b)
		}
	}
}

func TestParseBackendInvalid(t *testing.T) {
	_, err := ParseBackend("docker")
	if err == nil {
		t.Fatal("expected error for invalid backend")
	}
}

func TestBackendSupportsPlanTarget(t *testing.T) {
	tests := []struct {
		backend BackendID
		target  PlanTarget
		want    bool
	}{
		{BackendNanoVMS, PlanTargetNanoVMS, true},
		{BackendNanoVMS, PlanTargetDocker, false},
		{BackendPodman, PlanTargetDocker, true},
		{BackendPodman, PlanTargetNanoVMS, false},
		{BackendAppleContainers, PlanTargetDocker, true},
		{BackendWSLContainers, PlanTargetDocker, true},
		{"unknown", PlanTargetDocker, false},
	}
	for _, tt := range tests {
		if got := tt.backend.Supports(tt.target); got != tt.want {
			t.Errorf("backend=%s target=%s: got %v, want %v", tt.backend, tt.target, got, tt.want)
		}
	}
}

// --- Runtime Interface Tests ---

func TestTier1RuntimeInterface(t *testing.T) {
	rt := NewTier1Runtime()
	var _ Runtime = rt
	if rt.StartupTime() != 1*time.Millisecond {
		t.Errorf("expected 1ms startup, got %v", rt.StartupTime())
	}
}

func TestTier2RuntimeInterface(t *testing.T) {
	rt := NewTier2Runtime()
	var _ Runtime = rt
	if rt.StartupTime() != 90*time.Millisecond {
		t.Errorf("expected 90ms startup, got %v", rt.StartupTime())
	}
}

func TestTier3RuntimeInterface(t *testing.T) {
	rt := NewTier3Runtime()
	var _ Runtime = rt
	if rt.StartupTime() != 125*time.Millisecond {
		t.Errorf("expected 125ms startup, got %v", rt.StartupTime())
	}
}

// mockRuntime implements Runtime for testing.
type mockRuntime struct {
	name string
	tier int
}

func (m *mockRuntime) Name() string                   { return m.name }
func (m *mockRuntime) Tier() int                      { return m.tier }
func (m *mockRuntime) StartupTime() time.Duration     { return time.Millisecond }
func (m *mockRuntime) Deploy(ctx interface{}, config interface{}) (*domain.Sandbox, error) {
	return nil, nil
}
