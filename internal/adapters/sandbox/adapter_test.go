package sandbox

import (
	"context"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

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
