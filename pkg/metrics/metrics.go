// SPDX-License-Identifier: MIT OR Apache-2.0
// Package metrics provides a thread-safe collector for per-sandbox runtime
// metrics with Prometheus exposition format output.
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SandboxMetrics holds the runtime metrics for a single sandbox.
type SandboxMetrics struct {
	// CPU usage percentage (0.0 – 100.0+).
	CPU float64
	// Memory usage in megabytes.
	MemoryMB float64
	// Uptime is the duration the sandbox has been running.
	Uptime time.Duration
	// Status is a free-form label (e.g. "running", "stopped").
	Status string
}

// MetricsCollector is a thread-safe store of per-sandbox metrics.
// The zero value is NOT usable; always call NewMetricsCollector.
type MetricsCollector struct {
	mu       sync.Mutex
	sandboxes map[string]SandboxMetrics
}

// NewMetricsCollector returns an initialised MetricsCollector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		sandboxes: make(map[string]SandboxMetrics),
	}
}

// RecordSandbox stores (or overwrites) the metrics for the sandbox identified
// by id. An empty id is silently ignored.
func (m *MetricsCollector) RecordSandbox(id string, metrics SandboxMetrics) {
	if id == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxes[id] = metrics
}

// GetSandbox returns a pointer copy of the metrics for the given sandbox, or
// nil if no entry exists for id.
func (m *MetricsCollector) GetSandbox(id string) *SandboxMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	sm, ok := m.sandboxes[id]
	if !ok {
		return nil
	}
	return &sm
}

// AllMetrics returns a snapshot copy of every sandbox metric keyed by id.
func (m *MetricsCollector) AllMetrics() map[string]SandboxMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]SandboxMetrics, len(m.sandboxes))
	for k, v := range m.sandboxes {
		out[k] = v
	}
	return out
}

// RemoveSandbox deletes the metrics entry for the given sandbox id.
func (m *MetricsCollector) RemoveSandbox(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sandboxes, id)
}

// Count returns the number of tracked sandboxes.
func (m *MetricsCollector) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sandboxes)
}

// ToPrometheus produces a Prometheus text exposition format string containing
// HELP/TYPE metadata and per-sandbox labelled samples. The output is ordered
// by sandbox id for deterministic behaviour.
func (m *MetricsCollector) ToPrometheus() string {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sandboxes))
	for id := range m.sandboxes {
		ids = append(ids, id)
	}
	// Copy values under the lock so we can release it before formatting.
	type entry struct {
		sm SandboxMetrics
	}
	snapshot := make(map[string]entry, len(ids))
	for _, id := range ids {
		snapshot[id] = entry{sm: m.sandboxes[id]}
	}
	m.mu.Unlock()

	sort.Strings(ids)

	var b strings.Builder

	// -- CPU --
	b.WriteString("# HELP nvms_sandbox_cpu_percent CPU usage percentage per sandbox.\n")
	b.WriteString("# TYPE nvms_sandbox_cpu_percent gauge\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "nvms_sandbox_cpu_percent{sandbox=%q} %g\n", id, snapshot[id].sm.CPU)
	}

	// -- Memory --
	b.WriteString("# HELP nvms_sandbox_memory_mb Memory usage in megabytes per sandbox.\n")
	b.WriteString("# TYPE nvms_sandbox_memory_mb gauge\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "nvms_sandbox_memory_mb{sandbox=%q} %g\n", id, snapshot[id].sm.MemoryMB)
	}

	// -- Uptime --
	b.WriteString("# HELP nvms_sandbox_uptime_seconds Uptime in seconds per sandbox.\n")
	b.WriteString("# TYPE nvms_sandbox_uptime_seconds gauge\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "nvms_sandbox_uptime_seconds{sandbox=%q} %g\n", id, snapshot[id].sm.Uptime.Seconds())
	}

	// -- Status (mapped to numeric) --
	b.WriteString("# HELP nvms_sandbox_status Status code per sandbox (1=running, 0=stopped).\n")
	b.WriteString("# TYPE nvms_sandbox_status gauge\n")
	for _, id := range ids {
		val := 0.0
		if strings.EqualFold(snapshot[id].sm.Status, "running") {
			val = 1.0
		}
		fmt.Fprintf(&b, "nvms_sandbox_status{sandbox=%q, status=%q} %g\n",
			id, snapshot[id].sm.Status, val)
	}

	// -- Aggregate gauge: total tracked sandboxes --
	b.WriteString("# HELP nvms_sandbox_total Total number of tracked sandboxes.\n")
	b.WriteString("# TYPE nvms_sandbox_total gauge\n")
	fmt.Fprintf(&b, "nvms_sandbox_total %d\n", len(ids))

	return b.String()
}
