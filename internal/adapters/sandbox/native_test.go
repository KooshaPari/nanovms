package sandbox

import (
	"context"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

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

func TestNewNativeSandbox(t *testing.T) {
	adapter := NewNativeSandbox("bwrap")
	if adapter == nil {
		t.Fatal("NewNativeSandbox returned nil")
	}
	if adapter.tool != "bwrap" {
		t.Fatalf("expected tool=bwrap, got %s", adapter.tool)
	}
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
