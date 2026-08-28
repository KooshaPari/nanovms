// Package firecracker tests for the Firecracker adapter.
package firecracker

import (
	"context"
	"os/exec"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	if a.vms == nil {
		t.Fatal("vms map is nil")
	}
	if a.cmds == nil {
		t.Fatal("cmds map is nil")
	}
}

func TestCreate(t *testing.T) {
	a := NewAdapter()
	sb, err := a.Create(context.Background(), domain.SandboxConfig{Name: "test-vm"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb.ID == "" {
		t.Fatal("sandbox ID is empty")
	}
	if sb.Status != domain.SandboxStatusPending {
		t.Fatalf("expected %s, got %s", domain.SandboxStatusPending, sb.Status)
	}
	if sb.Type != domain.SandboxTypeVM {
		t.Fatalf("expected %s, got %s", domain.SandboxTypeVM, sb.Type)
	}
}

func TestGetNonexistent(t *testing.T) {
	a := NewAdapter()
	_, err := a.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent sandbox")
	}
}

func TestStart(t *testing.T) {
	if _, err := exec.LookPath("firecracker"); err != nil {
		t.Skip("firecracker not installed:", err)
	}
	a := NewAdapter()
	sb, _ := a.Create(context.Background(), domain.SandboxConfig{Name: "test"})
	if err := a.Start(context.Background(), sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = a.Stop(context.Background(), sb.ID, true) }()
	if sb.Status != domain.SandboxStatusRunning {
		t.Fatalf("expected running, got %s", sb.Status)
	}
}
