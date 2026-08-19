// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- adapter unit tests for Docker, gVisor, and Firecracker adapters.

package sandbox

import (
	"context"
	"testing"
)

// --- Interface compliance checks (compile-time) ---

var _ sandboxAdapter = (*DockerAdapter)(nil)
var _ sandboxAdapter = (*GVisorAdapter)(nil)
var _ sandboxAdapter = (*FirecrackerAdapter)(nil)

// sandboxAdapter is a minimal interface satisfied by all three adapters.
type sandboxAdapter interface {
	Name() string
	IsolationLevel() string
	Create(ctx context.Context, cfg SandboxConfig) (*SandboxResult, error)
	Stop(ctx context.Context, id SandboxID, force bool) error
	Remove(ctx context.Context, id SandboxID, force bool) error
	Status(ctx context.Context, id SandboxID) (*SandboxResult, error)
}

// --- Docker Adapter Tests ---

func TestDockerAdapterName(t *testing.T) {
	d := NewDockerAdapter()
	if d.Name() != "docker" {
		t.Fatalf("expected name "docker", got %q", d.Name())
	}
}

func TestDockerAdapterIsolationLevel(t *testing.T) {
	d := NewDockerAdapter()
	if d.IsolationLevel() != "container" {
		t.Fatalf("expected isolation "container", got %q", d.IsolationLevel())
	}
}

func TestDockerAdapterStopNonexistent(t *testing.T) {
	d := NewDockerAdapter()
	err := d.Stop(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Stop should fail for nonexistent container")
	}
}

func TestDockerAdapterRemoveNonexistent(t *testing.T) {
	d := NewDockerAdapter()
	err := d.Remove(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Remove should fail for nonexistent container")
	}
}

func TestDockerAdapterStatusNonexistent(t *testing.T) {
	d := NewDockerAdapter()
	_, err := d.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Status should fail for nonexistent container")
	}
}

// --- gVisor Adapter Tests ---

func TestGVisorAdapterName(t *testing.T) {
	g := NewGVisorAdapter()
	if g.Name() != "gvisor" {
		t.Fatalf("expected name "gvisor", got %q", g.Name())
	}
}

func TestGVisorAdapterIsolationLevel(t *testing.T) {
	g := NewGVisorAdapter()
	if g.IsolationLevel() != "container" {
		t.Fatalf("expected isolation "container", got %q", g.IsolationLevel())
	}
}

func TestGVisorAdapterStopNonexistent(t *testing.T) {
	g := NewGVisorAdapter()
	err := g.Stop(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Stop should fail for nonexistent container")
	}
}

func TestGVisorAdapterRemoveNonexistent(t *testing.T) {
	g := NewGVisorAdapter()
	err := g.Remove(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Remove should fail for nonexistent container")
	}
}

func TestGVisorAdapterStatusNonexistent(t *testing.T) {
	g := NewGVisorAdapter()
	_, err := g.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Status should fail for nonexistent container")
	}
}

// --- Firecracker Adapter Tests ---

func TestFirecrackerAdapterName(t *testing.T) {
	f := NewFirecrackerAdapter()
	if f.Name() != "firecracker" {
		t.Fatalf("expected name "firecracker", got %q", f.Name())
	}
}

func TestFirecrackerAdapterIsolationLevel(t *testing.T) {
	f := NewFirecrackerAdapter()
	if f.IsolationLevel() != "vm" {
		t.Fatalf("expected isolation "vm", got %q", f.IsolationLevel())
	}
}

func TestFirecrackerAdapterCreateAndStatus(t *testing.T) {
	f := NewFirecrackerAdapter()
	cfg := SandboxConfig{Name: "test-vm"}
	result, err := f.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if result.Status != SandboxStatusCreated {
		t.Fatalf("expected created status, got %s", result.Status)
	}

	status, err := f.Status(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Status != SandboxStatusCreated {
		t.Fatalf("expected created status from Status(), got %s", status.Status)
	}
}

func TestFirecrackerAdapterStopNonexistent(t *testing.T) {
	f := NewFirecrackerAdapter()
	err := f.Stop(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Stop should fail for nonexistent VM")
	}
}

func TestFirecrackerAdapterRemoveNonexistent(t *testing.T) {
	f := NewFirecrackerAdapter()
	err := f.Remove(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Remove should fail for nonexistent VM")
	}
}

func TestFirecrackerAdapterStatusNonexistent(t *testing.T) {
	f := NewFirecrackerAdapter()
	_, err := f.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Status should fail for nonexistent VM")
	}
}

func TestFirecrackerAdapterStopCreatedVM(t *testing.T) {
	f := NewFirecrackerAdapter()
	cfg := SandboxConfig{Name: "test-stop-vm"}
	result, _ := f.Create(context.Background(), cfg)

	err := f.Stop(context.Background(), result.ID, false)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	status, _ := f.Status(context.Background(), result.ID)
	if status.Status != SandboxStatusStopped {
		t.Fatalf("expected stopped status after Stop, got %s", status.Status)
	}
}

func TestFirecrackerAdapterRemoveCreatedVM(t *testing.T) {
	f := NewFirecrackerAdapter()
	cfg := SandboxConfig{Name: "test-remove-vm"}
	result, _ := f.Create(context.Background(), cfg)

	err := f.Remove(context.Background(), result.ID, false)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err = f.Status(context.Background(), result.ID)
	if err == nil {
		t.Fatal("Status should fail after Remove")
	}
}
