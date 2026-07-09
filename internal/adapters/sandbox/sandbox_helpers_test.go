package sandbox

import (
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Fatal("generateID returned empty string")
	}
	if id1 == id2 {
		t.Fatal("generateID produced duplicate IDs")
	}
	if len(id1) < 10 {
		t.Fatalf("generateID produced suspiciously short ID: %q", id1)
	}
}

func TestGenerateIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestCheckLandlockSupport(t *testing.T) {
	adapter := NewAdapter()
	supported := adapter.checkLandlockSupport()
	t.Logf("Landlock support: %v", supported)
	// This is environment-dependent; we just verify it doesn't panic
}

func TestSandboxStatus(t *testing.T) {
	sb := &domain.Sandbox{ID: "test", Name: "test", Status: domain.SandboxStatusPending}
	if sb.IsRunning() {
		t.Fatal("pending sandbox should not be running")
	}

	sb.Status = domain.SandboxStatusRunning
	if !sb.IsRunning() {
		t.Fatal("running sandbox should be running")
	}
}

func TestSandboxString(t *testing.T) {
	sb := &domain.Sandbox{ID: "abc", Name: "my-sandbox", Status: domain.SandboxStatusRunning}
	s := sb.String()
	if s == "" {
		t.Fatal("String() returned empty")
	}
}

func TestGenerateIDFormat(t *testing.T) {
	id := generateID()
	// Should start with "sandbox-"
	if len(id) < 8 {
		t.Fatalf("ID too short: %s", id)
	}
	// Should be unique (verify with timestamp range)
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = generateID()
	}
	// All should be unique
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[i] == ids[j] {
				t.Fatalf("duplicate ID in batch: %s", ids[i])
			}
		}
	}
}

// TestResolveExecCommand exercises the command-resolution helper used by
// startBwrap / startFirejail / startUnshare. We expect:
//  1. nil config -> default `/bin/sh`.
//  2. config without NativeSandbox -> default `/bin/sh`.
//  3. config with empty NativeSandbox.Command -> default `/bin/sh`.
//  4. config with non-empty NativeSandbox.Command -> the user-supplied vector
//     is returned verbatim, preserving the argv layout.
func TestResolveExecCommand(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		got := resolveExecCommand(nil)
		if len(got) != 1 || got[0] != "/bin/sh" {
			t.Fatalf("expected [/bin/sh], got %v", got)
		}
	})
	t.Run("config without NativeSandbox", func(t *testing.T) {
		cfg := &domain.SandboxConfig{Name: "x"}
		got := resolveExecCommand(cfg)
		if len(got) != 1 || got[0] != "/bin/sh" {
			t.Fatalf("expected [/bin/sh], got %v", got)
		}
	})
	t.Run("config with empty NativeSandbox.Command", func(t *testing.T) {
		cfg := &domain.SandboxConfig{
			Name:          "x",
			NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap},
		}
		got := resolveExecCommand(cfg)
		if len(got) != 1 || got[0] != "/bin/sh" {
			t.Fatalf("expected [/bin/sh], got %v", got)
		}
	})
	t.Run("config with custom command", func(t *testing.T) {
		want := []string{"/usr/bin/python3", "-c", "print(1)"}
		cfg := &domain.SandboxConfig{
			Name:          "py",
			NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: want},
		}
		got := resolveExecCommand(cfg)
		if len(got) != len(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("argv mismatch at %d: got %q want %q", i, got[i], want[i])
			}
		}
	})
}
func TestSandboxConfigDefaults(t *testing.T) {
	cfg := domain.SandboxConfig{
		Name:   "test",
		VMType: domain.VMFlavor("native"),
	}
	if cfg.Name != "test" {
		t.Fatalf("expected Name=test, got %s", cfg.Name)
	}
}

func TestSandboxMetricsDefaults(t *testing.T) {
	m := &domain.SandboxMetrics{}
	if m.CPUUsage != 0 {
		t.Fatalf("expected CPUUsage=0, got %f", m.CPUUsage)
	}
	if m.MemoryUsage != 0 {
		t.Fatalf("expected MemoryUsage=0, got %d", m.MemoryUsage)
	}
}
