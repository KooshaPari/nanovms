// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

// --- Engine Tests ---

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.tier1 == nil {
		t.Error("tier1 adapter not set")
	}
	if e.tier2 == nil {
		t.Error("tier2 adapter not set")
	}
	if e.tier3 == nil {
		t.Error("tier3 adapter not set")
	}
	if e.reservationTTL != 15*time.Minute {
		t.Errorf("expected 15m TTL, got %v", e.reservationTTL)
	}
}

func TestEngineDeployUnsupportedTier(t *testing.T) {
	e := NewEngine()
	_, err := e.Deploy(context.Background(), 99, domain.SandboxConfig{})
	if err == nil {
		t.Fatal("expected error for unsupported tier")
	}
}

func TestEngineStopUnsupportedTier(t *testing.T) {
	e := NewEngine()
	err := e.Stop(context.Background(), 99, "id-1")
	if err == nil {
		t.Fatal("expected error for unsupported tier")
	}
}

func TestEngineStopTier1(t *testing.T) {
	e := NewEngine()
	err := e.Stop(context.Background(), 1, "id-1")
	if err == nil {
		t.Fatal("expected error for tier1 stop")
	}
}

func TestEngineStopTier2(t *testing.T) {
	e := NewEngine()
	err := e.Stop(context.Background(), 2, "id-1")
	if err == nil {
		t.Fatal("expected error for tier2 stop")
	}
}

func TestEngineDeleteUnsupportedTier(t *testing.T) {
	e := NewEngine()
	err := e.Delete(context.Background(), 99, "id-1")
	if err == nil {
		t.Fatal("expected error for unsupported tier")
	}
}

func TestEngineDeleteTier1(t *testing.T) {
	e := NewEngine()
	err := e.Delete(context.Background(), 1, "id-1")
	if err == nil {
		t.Fatal("expected error for tier1 delete")
	}
}

func TestEngineDeleteTier2(t *testing.T) {
	e := NewEngine()
	err := e.Delete(context.Background(), 2, "id-1")
	if err == nil {
		t.Fatal("expected error for tier2 delete")
	}
}

func TestDeployFromConfigNil(t *testing.T) {
	e := NewEngine()
	_, err := e.DeployFromConfig(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

// --- mergeLabels Tests ---

func TestMergeLabelsBothNil(t *testing.T) {
	result := mergeLabels(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d", len(result))
	}
}

func TestMergeLabelsBaseNil(t *testing.T) {
	result := mergeLabels(nil, map[string]string{"a": "1"})
	if result["a"] != "1" {
		t.Errorf("expected a=1, got %s", result["a"])
	}
}

func TestMergeLabelsOverride(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	override := map[string]string{"b": "3", "c": "4"}
	result := mergeLabels(base, override)
	if result["a"] != "1" {
		t.Errorf("expected a=1, got %s", result["a"])
	}
	if result["b"] != "3" {
		t.Errorf("expected override b=3, got %s", result["b"])
	}
	if result["c"] != "4" {
		t.Errorf("expected c=4, got %s", result["c"])
	}
}

// --- EngineBackendDispatcher Tests ---

func TestRegisterBackendDispatcherNilEngine(t *testing.T) {
	var e *Engine
	err := e.RegisterBackendDispatcher(nvmsruntime.BackendPodman, &mockDispatcher{})
	if err == nil {
		t.Fatal("expected error for nil engine")
	}
}

func TestRegisterBackendDispatcherNilDispatcher(t *testing.T) {
	e := NewEngine()
	err := e.RegisterBackendDispatcher(nvmsruntime.BackendPodman, nil)
	if err == nil {
		t.Fatal("expected error for nil dispatcher")
	}
}

func TestRegisterBackendDispatcherInvalidBackend(t *testing.T) {
	e := NewEngine()
	err := e.RegisterBackendDispatcher("nonexistent", &mockDispatcher{})
	if err == nil {
		t.Fatal("expected error for invalid backend")
	}
}

func TestRegisterBackendDispatcherSuccess(t *testing.T) {
	e := NewEngine()
	d := &mockDispatcher{}
	err := e.RegisterBackendDispatcher(nvmsruntime.BackendPodman, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ConfigureGPUReservations Tests ---

func TestConfigureGPUReservationsNil(t *testing.T) {
	e := NewEngine()
	err := e.ConfigureGPUReservations(nil, 0)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

// mockDispatcher implements BackendDispatcher for testing.
type mockDispatcher struct{}

func (m *mockDispatcher) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	return &domain.Sandbox{ID: "mock-id"}, nil
}
