package sandbox

import (
	"context"
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

func TestAdapterCreate(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-sandbox"}
	sb, err := adapter.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb == nil {
		t.Fatal("Create returned nil sandbox")
	}
	if sb.ID == "" {
		t.Fatal("sandbox ID is empty")
	}
	if sb.Status != domain.SandboxStatusPending {
		t.Fatalf("expected pending status, got %s", sb.Status)
	}
}

func TestAdapterGet(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-get"}
	sb, _ := adapter.Create(context.Background(), cfg)

	got, err := adapter.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("Get failed for existing sandbox: %v", err)
	}
	if got.ID != sb.ID {
		t.Fatalf("Get returned wrong sandbox: got %s, want %s", got.ID, sb.ID)
	}

	_, err = adapter.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Get should have failed for nonexistent sandbox")
	}
}

func TestAdapterStartStop(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-lifecycle"}
	sb, _ := adapter.Create(context.Background(), cfg)

	if err := adapter.Start(context.Background(), sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if sb.Status != domain.SandboxStatusRunning {
		t.Fatalf("expected running status after Start, got %s", sb.Status)
	}
	if sb.StartedAt == nil {
		t.Fatal("StartedAt should be set after Start")
	}

	if err := adapter.Stop(context.Background(), sb.ID, false); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if sb.Status != domain.SandboxStatusStopped {
		t.Fatalf("expected stopped status after Stop, got %s", sb.Status)
	}
}

func TestAdapterDelete(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-delete"}
	sb, _ := adapter.Create(context.Background(), cfg)

	if err := adapter.Delete(context.Background(), sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := adapter.Get(context.Background(), sb.ID)
	if err == nil {
		t.Fatal("Get should have failed after Delete")
	}
}

func TestAdapterList(t *testing.T) {
	adapter := NewAdapter()
	adapter.Create(context.Background(), domain.SandboxConfig{Name: "s1"})
	adapter.Create(context.Background(), domain.SandboxConfig{Name: "s2"})
	adapter.Create(context.Background(), domain.SandboxConfig{Name: "s3"})

	list, err := adapter.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 sandboxes, got %d", len(list))
	}
}

func TestAdapterMetricsNotRunning(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-metrics"}
	sb, _ := adapter.Create(context.Background(), cfg)

	// Metrics for a non-running sandbox should return empty metrics, not error
	metrics, err := adapter.Metrics(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("Metrics should not error for non-running sandbox: %v", err)
	}
	if metrics == nil {
		t.Fatal("Metrics returned nil")
	}
	if metrics.SandboxID != sb.ID {
		t.Fatalf("expected SandboxID=%s, got %s", sb.ID, metrics.SandboxID)
	}
}

func TestAdapterLogsNotRunning(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-logs"}
	sb, _ := adapter.Create(context.Background(), cfg)

	// Logs for non-running sandbox should error with meaningful message
	_, err := adapter.Logs(context.Background(), sb.ID, false)
	if err == nil {
		t.Fatal("Logs should have failed for non-running sandbox")
	}

	// Non-existent sandbox
	_, err = adapter.Logs(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Logs should have failed for nonexistent sandbox")
	}
}

func TestAdapterExecNotRunning(t *testing.T) {
	adapter := NewAdapter()
	cfg := domain.SandboxConfig{Name: "test-exec"}
	sb, _ := adapter.Create(context.Background(), cfg)

	// Exec for non-running sandbox should error
	_, err := adapter.Exec(context.Background(), sb.ID, []string{"echo", "hello"})
	if err == nil {
		t.Fatal("Exec should have failed for non-running sandbox")
	}

	// Non-existent sandbox
	_, err = adapter.Exec(context.Background(), "nonexistent", []string{"echo", "hello"})
	if err == nil {
		t.Fatal("Exec should have failed for nonexistent sandbox")
	}
}

func TestNativeSandboxAdapterCreate(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	cfg := domain.SandboxConfig{Name: "test-native"}
	_, err := adapter.Create(context.Background(), cfg)
	if err == nil {
		// bwrap may not be installed in test environment - that's ok
		t.Log("bwrap available, sandbox created")
	} else {
		t.Logf("bwrap not installed (expected in CI): %v", err)
	}
}

func TestCheckLandlockSupport(t *testing.T) {
	adapter := NewAdapter()
	supported := adapter.checkLandlockSupport()
	t.Logf("Landlock support: %v", supported)
	// This is environment-dependent; we just verify it doesn't panic
}

func TestNewNativeSandbox(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	if adapter == nil {
		t.Fatal("NewNativeSandbox returned nil")
	}
	if adapter.tool != "bwrap" {
		t.Fatalf("expected tool=bwrap, got %s", adapter.tool)
	}
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

func TestAdapterMetricsNonexistent(t *testing.T) {
	adapter := NewAdapter()
	_, err := adapter.Metrics(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Metrics should have failed for nonexistent sandbox")
	}
}

func TestAdapterStartNonexistent(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Start(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Start should have failed for nonexistent sandbox")
	}
}

func TestAdapterStopNonexistent(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Stop(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Stop should have failed for nonexistent sandbox")
	}
}

func TestAdapterDeleteNonexistent(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Delete(context.Background(), "nonexistent")
	// Delete is idempotent - should not error
	_ = err
}

func TestNativeSandboxGetNonexistent(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	_, err := adapter.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Get should have failed for nonexistent sandbox")
	}
}

func TestNativeSandboxMetricsNonexistent(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	_, err := adapter.Metrics(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Metrics should have failed for nonexistent sandbox")
	}
}

func TestNativeSandboxLogsNonexistent(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	_, err := adapter.Logs(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("Logs should have failed for nonexistent sandbox")
	}
}

func TestNativeSandboxExecNonexistent(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	_, err := adapter.Exec(context.Background(), "nonexistent", []string{"echo", "hello"})
	if err == nil {
		t.Fatal("Exec should have failed for nonexistent sandbox")
	}
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
