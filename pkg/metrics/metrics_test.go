// SPDX-License-Identifier: MIT OR Apache-2.0
package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
	if mc.Count() != 0 {
		t.Fatalf("expected 0 sandboxes, got %d", mc.Count())
	}
}

func TestRecordAndGetSandbox(t *testing.T) {
	mc := NewMetricsCollector()
	sm := SandboxMetrics{CPU: 42.5, MemoryMB: 512, Uptime: 5 * time.Minute, Status: "running"}
	mc.RecordSandbox("sb-1", sm)

	got := mc.GetSandbox("sb-1")
	if got == nil {
		t.Fatal("GetSandbox returned nil for existing sandbox")
	}
	if got.CPU != 42.5 {
		t.Errorf("CPU = %g, want 42.5", got.CPU)
	}
	if got.MemoryMB != 512 {
		t.Errorf("MemoryMB = %g, want 512", got.MemoryMB)
	}
	if got.Uptime != 5*time.Minute {
		t.Errorf("Uptime = %v, want 5m", got.Uptime)
	}
	if got.Status != "running" {
		t.Errorf("Status = %q, want %q", got.Status, "running")
	}
}

func TestGetSandboxNotFound(t *testing.T) {
	mc := NewMetricsCollector()
	if got := mc.GetSandbox("nonexistent"); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestRecordSandboxEmptyId(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("", SandboxMetrics{CPU: 1.0})
	if mc.Count() != 0 {
		t.Fatal("empty id should not create an entry")
	}
}

func TestRecordSandboxOverwrite(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("sb-1", SandboxMetrics{CPU: 10})
	mc.RecordSandbox("sb-1", SandboxMetrics{CPU: 20})

	got := mc.GetSandbox("sb-1")
	if got == nil || got.CPU != 20 {
		t.Errorf("expected overwrite to CPU=20, got %v", got)
	}
	if mc.Count() != 1 {
		t.Fatalf("expected 1 sandbox after overwrite, got %d", mc.Count())
	}
}

func TestAllMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("a", SandboxMetrics{CPU: 1})
	mc.RecordSandbox("b", SandboxMetrics{CPU: 2})

	all := mc.AllMetrics()
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
	if all["a"].CPU != 1 || all["b"].CPU != 2 {
		t.Errorf("unexpected values: %v", all)
	}
}

func TestAllMetricsReturnsCopy(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("x", SandboxMetrics{CPU: 5})

	all := mc.AllMetrics()
	all["x"] = SandboxMetrics{CPU: 999} // mutate the copy

	got := mc.GetSandbox("x")
	if got == nil || got.CPU != 5 {
		t.Error("AllMetrics should return a defensive copy")
	}
}

func TestRemoveSandbox(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("sb-1", SandboxMetrics{CPU: 10})
	mc.RemoveSandbox("sb-1")

	if mc.GetSandbox("sb-1") != nil {
		t.Error("expected nil after RemoveSandbox")
	}
	if mc.Count() != 0 {
		t.Errorf("expected 0, got %d", mc.Count())
	}
}

func TestRemoveSandboxNonexistent(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RemoveSandbox("nope") // should not panic
}

func TestCount(t *testing.T) {
	mc := NewMetricsCollector()
	for i := 0; i < 5; i++ {
		mc.RecordSandbox(string(rune('a'+i)), SandboxMetrics{})
	}
	if mc.Count() != 5 {
		t.Errorf("expected 5, got %d", mc.Count())
	}
}

func TestToPrometheusEmpty(t *testing.T) {
	mc := NewMetricsCollector()
	out := mc.ToPrometheus()
	if out == "" {
		t.Fatal("ToPrometheus should not return empty string")
	}
	// Should still have the aggregate metric.
	if !strings.Contains(out, "nvms_sandbox_total 0") {
		t.Errorf("expected nvms_sandbox_total 0 in output:\n%s", out)
	}
}

func TestToPrometheusWithData(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("alpha", SandboxMetrics{
		CPU:      25.5,
		MemoryMB: 1024,
		Uptime:   2 * time.Minute,
		Status:   "running",
	})
	mc.RecordSandbox("beta", SandboxMetrics{
		CPU:      0.1,
		MemoryMB: 64,
		Uptime:   30 * time.Second,
		Status:   "stopped",
	})

	out := mc.ToPrometheus()

	// Verify key lines are present.
	checks := []string{
		"nvms_sandbox_cpu_percent{sandbox=\"alpha\"} 25.5",
		"nvms_sandbox_cpu_percent{sandbox=\"beta\"} 0.1",
		"nvms_sandbox_memory_mb{sandbox=\"alpha\"} 1024",
		"nvms_sandbox_memory_mb{sandbox=\"beta\"} 64",
		"nvms_sandbox_uptime_seconds{sandbox=\"alpha\"} 120",
		"nvms_sandbox_uptime_seconds{sandbox=\"beta\"} 30",
		"nvms_sandbox_status{sandbox=\"alpha\", status=\"running\"} 1",
		"nvms_sandbox_status{sandbox=\"beta\", status=\"stopped\"} 0",
		"nvms_sandbox_total 2",
		"# TYPE nvms_sandbox_cpu_percent gauge",
		"# HELP nvms_sandbox_memory_mb",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("missing expected line %q in output:\n%s", c, out)
		}
	}
}

func TestToPrometheusDeterministicOrder(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordSandbox("z", SandboxMetrics{})
	mc.RecordSandbox("a", SandboxMetrics{})
	mc.RecordSandbox("m", SandboxMetrics{})

	out1 := mc.ToPrometheus()
	out2 := mc.ToPrometheus()
	if out1 != out2 {
		t.Error("ToPrometheus should be deterministic")
	}

	// Verify ordering: 'a' before 'm' before 'z'.
	idxA := strings.Index(out1, "sandbox=\"a\"")
	idxM := strings.Index(out1, "sandbox=\"m\"")
	idxZ := strings.Index(out1, "sandbox=\"z\"")
	if !(idxA < idxM && idxM < idxZ) {
		t.Error("expected alphabetical ordering by sandbox id")
	}
}
