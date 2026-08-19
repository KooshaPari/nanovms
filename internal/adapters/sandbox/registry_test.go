// SPDX-License-Identifier: MIT OR Apache-2.0
package sandbox

import (
	"testing"
)

func TestNewAdapterRegistry(t *testing.T) {
	r := NewAdapterRegistry()
	if r == nil {
		t.Fatal("NewAdapterRegistry returned nil")
	}
	list := r.List()
	if len(list) != 11 {
		t.Fatalf("expected 11 built-in adapters, got %d", len(list))
	}
}

func TestAdapterRegistryGet(t *testing.T) {
	r := NewAdapterRegistry()
	cap, err := r.Get(AdapterDocker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.Type != AdapterDocker {
		t.Errorf("expected docker, got %s", cap.Type)
	}
	if cap.IsolationLevel != "container" {
		t.Errorf("expected container isolation, got %s", cap.IsolationLevel)
	}
}

func TestAdapterRegistryGetNotFound(t *testing.T) {
	r := NewAdapterRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown adapter")
	}
}

func TestAdapterRegistryRegister(t *testing.T) {
	r := NewAdapterRegistry()
	r.Register(AdapterCapabilities{
		Type:           "custom",
		SupportsGPU:    true,
		MaxMemoryMB:    1024,
		MaxCPU:         4,
		IsolationLevel: "vm",
	})
	cap, err := r.Get("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cap.SupportsGPU {
		t.Error("expected GPU support")
	}
}

func TestSelectBestNoGPU(t *testing.T) {
	r := NewAdapterRegistry()
	cap, err := r.SelectBest(false, 512, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.IsolationLevel != "vm" {
		t.Errorf("expected VM isolation for best fit, got %s", cap.IsolationLevel)
	}
}

func TestSelectBestWithGPU(t *testing.T) {
	r := NewAdapterRegistry()
	cap, err := r.SelectBest(true, 1024, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cap.SupportsGPU {
		t.Error("expected adapter with GPU support")
	}
}

func TestAdapterFirecrackerRequiresRoot(t *testing.T) {
	r := NewAdapterRegistry()
	cap, _ := r.Get(AdapterFirecracker)
	if !cap.RequiresRoot {
		t.Error("firecracker should require root")
	}
	if cap.IsolationLevel != "vm" {
		t.Errorf("expected vm isolation, got %s", cap.IsolationLevel)
	}
}

func TestAdapterLinuxGPU(t *testing.T) {
	r := NewAdapterRegistry()
	cap, _ := r.Get(AdapterLinux)
	if !cap.SupportsGPU {
		t.Error("linux adapter should support GPU")
	}
	if cap.IsolationLevel != "process" {
		t.Errorf("expected process isolation, got %s", cap.IsolationLevel)
	}
}
