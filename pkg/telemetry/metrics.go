// SPDX-License-Identifier: MIT OR Apache-2.0
// Package telemetry – metrics collection helpers.
//
// Provides predefined metrics (request count, latency, errors), histogram
// buckets, custom metric registration, and metric exporter utilities.
package telemetry

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Metric types
// ---------------------------------------------------------------------------

// MetricKind identifies the metric instrument kind.
type MetricKind string

const (
	KindCounter   MetricKind = "counter"
	KindGauge     MetricKind = "gauge"
	KindHistogram MetricKind = "histogram"
)

// ---------------------------------------------------------------------------
// Histogram buckets
// ---------------------------------------------------------------------------

// DefaultLatencyBuckets returns bucket boundaries (in milliseconds) suited
// for HTTP request latency measurement.
func DefaultLatencyBuckets() []float64 {
	return []float64{
		0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
	}
}

// DefaultSizeBuckets returns bucket boundaries for payload-size histograms
// measured in bytes.
func DefaultSizeBuckets() []float64 {
	return []float64{
		64, 128, 256, 512, 1024, 4096, 16384, 65536, 262144, 1048576,
	}
}

// DefaultErrorRateBuckets returns bucket boundaries for error-rate
// percentages [0, 100].
func DefaultErrorRateBuckets() []float64 {
	return []float64{0.1, 0.5, 1, 2, 5, 10, 20, 50, 80, 100}
}

// LinearBuckets generates count evenly-spaced buckets from start to end.
func LinearBuckets(start, end float64, count int) []float64 {
	if count <= 0 || end <= start {
		return nil
	}
	step := (end - start) / float64(count)
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start + float64(i+1)*step
	}
	return buckets
}

// ExponentialBuckets generates count exponentially-spaced buckets from
// start, each multiplied by factor.
func ExponentialBuckets(start, factor float64, count int) []float64 {
	if count <= 0 || start <= 0 || factor <= 1 {
		return nil
	}
	buckets := make([]float64, count)
	buckets[0] = start
	for i := 1; i < count; i++ {
		buckets[i] = buckets[i-1] * factor
	}
	return buckets
}

// ---------------------------------------------------------------------------
// Counter
// ---------------------------------------------------------------------------

// Counter is a thread-safe monotonic counter.
type Counter struct {
	name   string
	labels map[string]string
	value  atomic.Int64
}

// NewCounter creates a named counter with optional labels.
func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{name: name, labels: labels}
}

// Inc atomically increments the counter by 1.
func (c *Counter) Inc() { c.Add(1) }

// Add atomically increments the counter by delta.
func (c *Counter) Add(delta int64) { c.value.Add(delta) }

// Value returns the current counter value.
func (c *Counter) Value() int64 { return c.value.Load() }

// Name returns the metric name.
func (c *Counter) Name() string { return c.name }

// Labels returns the metric labels.
func (c *Counter) Labels() map[string]string { return c.labels }

// Reset sets the counter to zero.
func (c *Counter) Reset() { c.value.Store(0) }

// ---------------------------------------------------------------------------
// Gauge
// ---------------------------------------------------------------------------

// Gauge is a thread-safe gauge that can go up and down.
type Gauge struct {
	name   string
	labels map[string]string
	value  atomic.Int64
}

// NewGauge creates a named gauge with optional labels.
func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{name: name, labels: labels}
}

// Set atomically sets the gauge to the given value.
func (g *Gauge) Set(val int64) { g.value.Store(val) }

// Inc atomically increments the gauge by 1.
func (g *Gauge) Inc() { g.value.Add(1) }

// Dec atomically decrements the gauge by 1.
func (g *Gauge) Dec() { g.value.Add(-1) }

// Value returns the current gauge value.
func (g *Gauge) Value() int64 { return g.value.Load() }

// Name returns the metric name.
func (g *Gauge) Name() string { return g.name }

// Labels returns the metric labels.
func (g *Gauge) Labels() map[string]string { return g.labels }

// ---------------------------------------------------------------------------
// Histogram
// ---------------------------------------------------------------------------

// Histogram tracks value distributions with configurable bucket boundaries.
type Histogram struct {
	name    string
	labels  map[string]string
	buckets []float64
	counts  []int64
	sum     atomic.Int64 // stored as int64 with microsecond precision
	count   atomic.Int64
}

// NewHistogram creates a named histogram with the given bucket boundaries.
func NewHistogram(name string, labels map[string]string, buckets []float64) *Histogram {
	if len(buckets) == 0 {
		buckets = DefaultLatencyBuckets()
	}
	sorted := make([]float64, len(buckets))
	copy(sorted, buckets)
	sort.Float64s(sorted)

	return &Histogram{
		name:    name,
		labels:  labels,
		buckets: sorted,
		counts:  make([]int64, len(sorted)+1), // +1 for +Inf bucket
	}
}

// Observe records a value into the histogram.
func (h *Histogram) Observe(value float64) {
	h.count.Add(1)
	h.sum.Add(int64(value * 1000)) // microsecond precision

	idx := sort.SearchFloat64s(h.buckets, value)
	if idx < len(h.buckets) && h.buckets[idx] < value {
		idx++
	}
	h.counts[idx]++
}

// Count returns the total number of observations.
func (h *Histogram) Count() int64 { return h.count.Load() }

// Sum returns the sum of all observations.
func (h *Histogram) Sum() float64 { return float64(h.sum.Load()) / 1000.0 }

// Mean returns the average of all observations.
func (h *Histogram) Mean() float64 {
	c := h.Count()
	if c == 0 {
		return 0
	}
	return h.Sum() / float64(c)
}

// Buckets returns a copy of the bucket boundaries and their cumulative counts.
func (h *Histogram) Buckets() ([]float64, []int64) {
	counts := make([]int64, len(h.counts))
	copy(counts, h.counts)
	return h.buckets, counts
}

// Percentile estimates the p-th percentile (0–100) from bucket data.
// This is an approximation based on histogram buckets.
func (h *Histogram) Percentile(p float64) float64 {
	c := h.Count()
	if c == 0 {
		return 0
	}
	target := int64(math.Ceil(p / 100.0 * float64(c)))
	var cumulative int64
	for i, cnt := range h.counts {
		cumulative += cnt
		if cumulative >= target {
			if i < len(h.buckets) {
				return h.buckets[i]
			}
			return math.MaxFloat64
		}
	}
	return math.MaxFloat64
}

// Name returns the metric name.
func (h *Histogram) Name() string { return h.name }

// Labels returns the metric labels.
func (h *Histogram) Labels() map[string]string { return h.labels }

// Reset clears all histogram data.
func (h *Histogram) Reset() {
	h.count.Store(0)
	h.sum.Store(0)
	for i := range h.counts {
		h.counts[i] = 0
	}
}

// ---------------------------------------------------------------------------
// Registry – central metric registry
// ---------------------------------------------------------------------------

// Registry holds all registered metrics and provides lookup/iteration.
type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	histos   map[string]*Histogram
}

// NewRegistry creates an empty metric registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
		histos:   make(map[string]*Histogram),
	}
}

func metricKey(name string, labels map[string]string) string {
	key := name
	// Deterministic key from labels.
	if len(labels) > 0 {
		keys := make([]string, 0, len(labels))
		for k := range labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			key += fmt.Sprintf("|%s=%s", k, labels[k])
		}
	}
	return key
}

// RegisterCounter registers or returns an existing counter.
func (r *Registry) RegisterCounter(name string, labels map[string]string) *Counter {
	key := metricKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[key]; ok {
		return c
	}
	c := NewCounter(name, labels)
	r.counters[key] = c
	return c
}

// RegisterGauge registers or returns an existing gauge.
func (r *Registry) RegisterGauge(name string, labels map[string]string) *Gauge {
	key := metricKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.gauges[key]; ok {
		return g
	}
	g := NewGauge(name, labels)
	r.gauges[key] = g
	return g
}

// RegisterHistogram registers or returns an existing histogram.
func (r *Registry) RegisterHistogram(name string, labels map[string]string, buckets []float64) *Histogram {
	key := metricKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histos[key]; ok {
		return h
	}
	h := NewHistogram(name, labels, buckets)
	r.histos[key] = h
	return h
}

// AllCounters returns all registered counters.
func (r *Registry) AllCounters() []*Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Counter, 0, len(r.counters))
	for _, c := range r.counters {
		out = append(out, c)
	}
	return out
}

// AllGauges returns all registered gauges.
func (r *Registry) AllGauges() []*Gauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Gauge, 0, len(r.gauges))
	for _, g := range r.gauges {
		out = append(out, g)
	}
	return out
}

// AllHistograms returns all registered histograms.
func (r *Registry) AllHistograms() []*Histogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Histogram, 0, len(r.histos))
	for _, h := range r.histos {
		out = append(out, h)
	}
	return out
}

// Reset clears all metrics in the registry.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.counters {
		c.Reset()
	}
	for _, g := range r.gauges {
		g.Set(0)
	}
	for _, h := range r.histos {
		h.Reset()
	}
}

// ---------------------------------------------------------------------------
// Meter – combines registry + push loop
// ---------------------------------------------------------------------------

// Meter ties a registry to an exporter, periodically pushing metric snapshots.
type Meter struct {
	registry *Registry
	service  string
	config   MetricsConfig
	stopCh   chan struct{}
}

// NewMeter creates a Meter with the given configuration.
func NewMeter(service string, cfg MetricsConfig) *Meter {
	return &Meter{
		registry: NewRegistry(),
		service:  service,
		config:   cfg,
		stopCh:   make(chan struct{}),
	}
}

// Registry returns the metric registry.
func (m *Meter) Registry() *Registry { return m.registry }

// StartPushLoop periodically exports metrics. It blocks until the context is
// cancelled or Shutdown is called.
func (m *Meter) StartPushLoop(ctx context.Context, exporter *Exporter, interval time.Duration) {
	if interval <= 0 {
		interval = m.config.Interval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			_ = m.push(ctx, exporter)
		}
	}
}

// push collects all metrics and exports them.
func (m *Meter) push(ctx context.Context, exporter *Exporter) error {
	metrics := make([]MetricData, 0)

	for _, c := range m.registry.AllCounters() {
		metrics = append(metrics, MetricData{
			Name:      c.Name(),
			Type:      string(KindCounter),
			Value:     float64(c.Value()),
			Timestamp: time.Now(),
			Labels:    c.Labels(),
		})
	}

	for _, g := range m.registry.AllGauges() {
		metrics = append(metrics, MetricData{
			Name:      g.Name(),
			Type:      string(KindGauge),
			Value:     float64(g.Value()),
			Timestamp: time.Now(),
			Labels:    g.Labels(),
		})
	}

	for _, h := range m.registry.AllHistograms() {
		metrics = append(metrics, MetricData{
			Name:      h.Name(),
			Type:      string(KindHistogram),
			Value:     h.Mean(),
			Timestamp: time.Now(),
			Labels:    h.Labels(),
		})
	}

	return exporter.ExportMetrics(ctx, metrics)
}

// Shutdown stops the push loop and flushes remaining data.
func (m *Meter) Shutdown(_ context.Context) {
	close(m.stopCh)
}

// ---------------------------------------------------------------------------
// Predefined nanovms metrics
// ---------------------------------------------------------------------------

// Predefined metric names used across the platform.
const (
	MetricRequestCount      = "nanovms.requests.total"
	MetricRequestLatency    = "nanovms.requests.latency_ms"
	MetricErrorCount        = "nanovms.errors.total"
	MetricSandboxDeploys    = "nanovms.sandboxes.deploys"
	MetricSandboxStops      = "nanovms.sandboxes.stops"
	MetricSandboxErrors     = "nanovms.sandboxes.errors"
	MetricVMBootLatency     = "nanovms.vm.boot_ms"
	MetricContainerLifecycle = "nanovms.container.lifecycle_ms"
	MetricQueueDepth        = "nanovms.queue.depth"
	MetricActiveConnections = "nanovms.connections.active"
	MetricGC Pause          = "nanovms.gc.pause_ms"
)

// PredefinedLabels is a set of common label keys.
var PredefinedLabels = struct {
	Method   string
	Path     string
	Status   string
	Region   string
	Tier     string
	Protocol string
}{
	Method:   "method",
	Path:     "path",
	Status:   "status",
	Region:   "region",
	Tier:     "tier",
	Protocol: "protocol",
}

// SetupNanovmsMetrics registers the full set of predefined nanovms metrics
// on a registry and returns handles for easy use.
func SetupNanovmsMetrics(reg *Registry) *NanovmsMetrics {
	return &NanovmsMetrics{
		RequestCount:      reg.RegisterCounter(MetricRequestCount, nil),
		RequestLatency:    reg.RegisterHistogram(MetricRequestLatency, nil, DefaultLatencyBuckets()),
		ErrorCount:        reg.RegisterCounter(MetricErrorCount, nil),
		SandboxDeploys:    reg.RegisterCounter(MetricSandboxDeploys, nil),
		SandboxStops:      reg.RegisterCounter(MetricSandboxStops, nil),
		SandboxErrors:     reg.RegisterCounter(MetricSandboxErrors, nil),
		VMBootLatency:     reg.RegisterHistogram(MetricVMBootLatency, nil, DefaultLatencyBuckets()),
		QueueDepth:        reg.RegisterGauge(MetricQueueDepth, nil),
		ActiveConnections: reg.RegisterGauge(MetricActiveConnections, nil),
	}
}

// NanovmsMetrics holds predefined metric handles for convenient access.
type NanovmsMetrics struct {
	RequestCount      *Counter
	RequestLatency    *Histogram
	ErrorCount        *Counter
	SandboxDeploys    *Counter
	SandboxStops      *Counter
	SandboxErrors     *Counter
	VMBootLatency     *Histogram
	QueueDepth        *Gauge
	ActiveConnections *Gauge
}

// RecordRequest is a convenience method that increments the request counter
// and records latency in one call.
func (nm *NanovmsMetrics) RecordRequest(method, path, status string, latencyMs float64) {
	nm.RequestCount.Inc()
	nm.RequestLatency.Observe(latencyMs)
	if status[0] >= '4' { // 4xx or 5xx
		nm.ErrorCount.Inc()
	}
}

// ---------------------------------------------------------------------------
// Metric exporter utilities
// ---------------------------------------------------------------------------

// MetricSnapshot is a point-in-time snapshot of a single metric.
type MetricSnapshot struct {
	Name      string            `json:"name"`
	Kind      MetricKind        `json:"kind"`
	Value     float64           `json:"value"`
	Count     int64             `json:"count"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// CollectSnapshots captures the current state of all metrics in a registry.
func CollectSnapshots(reg *Registry) []MetricSnapshot {
	now := time.Now()
	snaps := make([]MetricSnapshot, 0)

	for _, c := range reg.AllCounters() {
		snaps = append(snaps, MetricSnapshot{
			Name:      c.Name(),
			Kind:      KindCounter,
			Value:     float64(c.Value()),
			Labels:    c.Labels(),
			Timestamp: now,
		})
	}

	for _, g := range reg.AllGauges() {
		snaps = append(snaps, MetricSnapshot{
			Name:      g.Name(),
			Kind:      KindGauge,
			Value:     float64(g.Value()),
			Labels:    g.Labels(),
			Timestamp: now,
		})
	}

	for _, h := range reg.AllHistograms() {
		snaps = append(snaps, MetricSnapshot{
			Name:      h.Name(),
			Kind:      KindHistogram,
			Value:     h.Mean(),
			Count:     h.Count(),
			Labels:    h.Labels(),
			Timestamp: now,
		})
	}

	return snaps
}
