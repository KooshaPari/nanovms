//go:build integration

// Package e2e contains real end-to-end integration tests for the nanovms
// tier execution paths. These tests launch actual runtimes and are only
// compiled/run with the `integration` build tag; each test skips itself
// cleanly when its underlying runtime binary is not present on PATH so the
// suite stays green on hosts without the toolchain installed.
package e2e

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters/gvisor"
	"github.com/kooshapari/nanovms/internal/domain"
)

// runscAvailable reports whether the gVisor runsc binary can be found on
// PATH. Callers use it to gate the real-lifecycle tests.
func runscAvailable() bool {
	_, err := exec.LookPath("runsc")
	return err == nil
}

// TestGVisorE2EFullLifecycle exercises the gVisor adapter's real runsc
// lifecycle end to end: Create -> Start -> Stop -> Delete. The test is
// skipped when runsc is not installed (e.g. typical CI boxes, where gVisor
// is unavailable). When runsc IS available it verifies the sandbox's status
// transitions from pending -> running and that the sandbox becomes
// unlisted after Delete.
func TestGVisorE2EFullLifecycle(t *testing.T) {
	if !runscAvailable() {
		t.Skip("runsc binary not found on PATH; skipping gVisor e2e integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := gvisor.NewAdapter()

	// Create: allocate a pending sandbox descriptor.
	sb, err := adapter.Create(ctx, domain.SandboxConfig{Name: "e2e-gvisor"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb.ID == "" {
		t.Fatal("Create returned empty sandbox ID")
	}
	if sb.Status != domain.SandboxStatusPending {
		t.Fatalf("after Create expected status %q, got %q", domain.SandboxStatusPending, sb.Status)
	}
	if sb.Type != domain.SandboxTypeGVisor {
		t.Fatalf("expected type %q, got %q", domain.SandboxTypeGVisor, sb.Type)
	}

	// Start: launch the runsc sandbox; status must become running.
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	got, err := adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get after Start failed: %v", err)
	}
	if got.Status != domain.SandboxStatusRunning {
		t.Fatalf("after Start expected status %q, got %q", domain.SandboxStatusRunning, got.Status)
	}

	// Stop: terminate the running sandbox.
	if err := adapter.Stop(ctx, sb.ID, true); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Delete: remove the sandbox entirely; it must no longer be listed.
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	list, err := adapter.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	for _, s := range list {
		if s.ID == sb.ID {
			t.Fatalf("sandbox %s still present after Delete", sb.ID)
		}
	}
}

// TestGVisorE2EExec runs a trivial command inside a real runsc sandbox to
// exercise the Exec path. It is gated on runsc availability just like the
// full lifecycle test.
func TestGVisorE2EExec(t *testing.T) {
	if !runscAvailable() {
		t.Skip("runsc binary not found on PATH; skipping gVisor e2e exec test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := gvisor.NewAdapter()
	sb, err := adapter.Create(ctx, domain.SandboxConfig{Name: "e2e-gvisor-exec"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = adapter.Delete(ctx, sb.ID) }()

	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	rc, err := adapter.Exec(ctx, sb.ID, []string{"/bin/echo", "nanovms-e2e"})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if rc != nil {
		_ = rc.Close()
	}
}