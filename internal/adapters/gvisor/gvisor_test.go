// Package gvisor tests for the gVisor adapter.
package gvisor

import (
	"context"
	"os/exec"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	if a == nil {
		t.Fatal("NewAdapter returned nil")
	}
	if a.sandboxes == nil {
		t.Fatal("sandboxes map is nil")
	}
	if a.cmds == nil {
		t.Fatal("cmds map is nil")
	}
}

func TestCreate(t *testing.T) {
	a := NewAdapter()
	sb, err := a.Create(context.Background(), domain.SandboxConfig{Name: "test-sb"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb.ID == "" {
		t.Fatal("sandbox ID is empty")
	}
	if sb.Status != domain.SandboxStatusPending {
		t.Fatalf("expected %s, got %s", domain.SandboxStatusPending, sb.Status)
	}
	if sb.Type != domain.SandboxTypeGVisor {
		t.Fatalf("expected %s, got %s", domain.SandboxTypeGVisor, sb.Type)
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
	if _, err := exec.LookPath("runsc"); err != nil {
		t.Skip("runsc not installed:", err)
	}
	a := NewAdapter()
	sb, _ := a.Create(context.Background(), domain.SandboxConfig{Name: "test"})
	if err := a.Start(context.Background(), sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer a.Stop(context.Background(), sb.ID, true)
	if sb.Status != domain.SandboxStatusRunning {
		t.Fatalf("expected running, got %s", sb.Status)
	}
}
