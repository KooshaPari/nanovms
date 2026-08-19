// SPDX-License-Identifier: MIT OR Apache-2.0
package metrics

import (
	"testing"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
}

func TestCollectorRecordAndGet(t *testing.T) {
	c := NewCollector()
	c.Record("test.metric", MetricGauge, 42.0, map[string]string{"env": "test"})
	metrics := c.Get("test.metric")
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Value != 42.0 {
		t.Errorf("expected 42.0, got %f", metrics[0].Value)
	}
}

func TestCollectorGetEmpty(t *testing.T) {
	c := NewCollector()
	metrics := c.Get("nonexistent")
	if len(metrics) != 0 {
		t.Errorf("expected empty, got %d", len(metrics))
	}
}

func TestCollectorSummary(t *testing.T) {
	c := NewCollector()
	c.Record("latency", MetricHistogram, 10.0, nil)
	c.Record("latency", MetricHistogram, 20.0, nil)
	c.Record("latency", MetricHistogram, 30.0, nil)

	count, sum, min, max := c.Summary("latency")
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
	if sum != 60.0 {
		t.Errorf("expected sum 60, got %f", sum)
	}
	if min != 10.0 {
		t.Errorf("expected min 10, got %f", min)
	}
	if max != 30.0 {
		t.Errorf("expected max 30, got %f", max)
	}
}

func TestCollectorSummaryEmpty(t *testing.T) {
	c := NewCollector()
	count, sum, min, max := c.Summary("empty")
	if count != 0 || sum != 0 || min != 0 || max != 0 {
		t.Error("expected all zeros for empty summary")
	}
}

func TestCollectorReset(t *testing.T) {
	c := NewCollector()
	c.Record("test", MetricCounter, 1.0, nil)
	c.Reset()
	metrics := c.Get("test")
	if len(metrics) != 0 {
		t.Errorf("expected empty after reset, got %d", len(metrics))
	}
}

func TestCollectorPrintSummary(t *testing.T) {
	c := NewCollector()
	c.Record("test", MetricGauge, 5.0, nil)
	c.Record("test", MetricGauge, 15.0, nil)
	result := c.PrintSummary("test")
	if result == "" {
		t.Error("expected non-empty summary")
	}
}

func TestCollectorPrintSummaryEmpty(t *testing.T) {
	c := NewCollector()
	result := c.PrintSummary("missing")
	if result != "missing: no data" {
		t.Errorf("expected 'missing: no data', got %s", result)
	}
}

func TestSandboxMetricsRecordDeploy(t *testing.T) {
	sm := NewSandboxMetrics()
	sm.RecordDeploy(100.0, 2, true)
	if sm.Deploys != 1 {
		t.Errorf("expected 1 deploy, got %d", sm.Deploys)
	}
	if sm.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", sm.Errors)
	}
}

func TestSandboxMetricsRecordDeployError(t *testing.T) {
	sm := NewSandboxMetrics()
	sm.RecordDeploy(50.0, 1, false)
	if sm.Deploys != 1 {
		t.Errorf("expected 1 deploy, got %d", sm.Deploys)
	}
	if sm.Errors != 1 {
		t.Errorf("expected 1 error, got %d", sm.Errors)
	}
}
