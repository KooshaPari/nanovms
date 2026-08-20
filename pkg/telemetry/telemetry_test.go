// SPDX-License-Identifier: MIT OR Apache-2.0
package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Config tests
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServiceName != "nanovms" {
		t.Errorf("expected service name 'nanovms', got %q", cfg.ServiceName)
	}
	if cfg.OTLP.Endpoint == "" {
		t.Error("expected non-empty OTLP endpoint")
	}
	if cfg.OTLP.Protocol != "grpc" {
		t.Errorf("expected protocol 'grpc', got %q", cfg.OTLP.Protocol)
	}
	if cfg.Metrics.Interval == 0 {
		t.Error("expected non-zero metrics interval")
	}
	if len(cfg.Metrics.HistogramBuckets) == 0 {
		t.Error("expected non-empty histogram buckets")
	}
	if cfg.Health.Port == 0 {
		t.Error("expected non-zero health port")
	}
	if len(cfg.Propagation.Formats) == 0 {
		t.Error("expected non-empty propagation formats")
	}
}

// ---------------------------------------------------------------------------
// Tracer tests
// ---------------------------------------------------------------------------

func TestTracer_StartEndSpan(t *testing.T) {
	tracer := NewTracer("test-service", SamplingConfig{Strategy: "always"})
	ctx := tracer.StartSpan(context.Background(), "test-operation", SpanKindServer)
	span := SpanFromContext(ctx)
	if span == nil {
		t.Fatal("expected span in context")
	}
	if span.TraceID == "" {
		t.Error("expected non-empty trace ID")
	}
	if span.SpanID == "" {
		t.Error("expected non-empty span ID")
	}
	if span.Name != "test-operation" {
		t.Errorf("expected span name 'test-operation', got %q", span.Name)
	}
	if span.Kind != SpanKindServer {
		t.Errorf("expected SpanKindServer, got %d", span.Kind)
	}

	tracer.EndSpan(span)
	if span.EndTime.IsZero() {
		t.Error("expected non-zero end time")
	}
	if span.EndTime.Before(span.StartTime) {
		t.Error("end time should not be before start time")
	}
}

func TestTracer_ChildSpan(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})

	parentCtx, parentSpan := tracer.StartSpan(context.Background(), "parent", SpanKindInternal)
	childCtx, childSpan := tracer.StartSpan(parentCtx, "child", SpanKindInternal)

	if childSpan.ParentID != parentSpan.SpanID {
		t.Errorf("expected child parent ID %q, got %q", parentSpan.SpanID, childSpan.ParentID)
	}
	if childSpan.TraceID != parentSpan.TraceID {
		t.Error("child should share trace ID with parent")
	}

	tracer.EndSpan(childSpan)
	tracer.EndSpan(parentSpan)

	if tracer.SpanCount() != 2 {
		t.Errorf("expected 2 spans, got %d", tracer.SpanCount())
	}
}

func TestTracer_Sampling_Never(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "never"})
	for i := 0; i < 100; i++ {
		_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)
		if span.Sampled {
			t.Errorf("expected span %d to be unsampled", i)
		}
	}
}

func TestTracer_Sampling_Always(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	for i := 0; i < 100; i++ {
		_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)
		if !span.Sampled {
			t.Errorf("expected span %d to be sampled", i)
		}
	}
}

func TestTracer_AddEvent(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)

	tracer.AddEvent(span, "cache.miss", map[string]string{"key": "user:123"})
	tracer.AddEvent(span, "retry", nil)

	if len(span.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(span.Events))
	}
	if span.Events[0].Name != "cache.miss" {
		t.Errorf("expected event name 'cache.miss', got %q", span.Events[0].Name)
	}
	if span.Events[0].Attributes["key"] != "user:123" {
		t.Errorf("expected attribute key 'user:123', got %q", span.Events[0].Attributes["key"])
	}
}

func TestTracer_SetStatus(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)

	tracer.SetStatus(span, StatusCodeError, "something went wrong")

	if span.Status.Code != StatusCodeError {
		t.Errorf("expected StatusCodeError, got %d", span.Status.Code)
	}
	if span.Status.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", span.Status.Message)
	}
}

func TestTracer_Reset(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	for i := 0; i < 10; i++ {
		_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)
		tracer.EndSpan(span)
	}
	if tracer.SpanCount() != 10 {
		t.Fatalf("expected 10 spans before reset, got %d", tracer.SpanCount())
	}

	tracer.Reset()
	if tracer.SpanCount() != 0 {
		t.Errorf("expected 0 spans after reset, got %d", tracer.SpanCount())
	}
}

func TestTracer_SpanOptions(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal,
		WithAttributes(map[string]string{"env": "staging", "region": "us-east-1"}),
	)

	if span.Attributes["env"] != "staging" {
		t.Errorf("expected env=staging, got %q", span.Attributes["env"])
	}
	if span.Attributes["region"] != "us-east-1" {
		t.Errorf("expected region=us-east-1, got %q", span.Attributes["region"])
	}
}

func TestTracer_EndOptions(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)

	tracer.EndSpan(span, WithError(context.DeadlineExceeded))

	if span.Status.Code != StatusCodeError {
		t.Errorf("expected StatusCodeError, got %d", span.Status.Code)
	}
}

func TestTracer_ConcurrentAccess(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)
			tracer.AddEvent(span, "event", nil)
			tracer.EndSpan(span)
		}()
	}
	wg.Wait()
	if tracer.SpanCount() != 100 {
		t.Errorf("expected 100 spans, got %d", tracer.SpanCount())
	}
}

// ---------------------------------------------------------------------------
// Context propagation tests
// ---------------------------------------------------------------------------

func TestPropagator_InjectExtract(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	ctx, span := tracer.StartSpan(context.Background(), "op", SpanKindServer)

	prop := NewPropagator([]string{"tracecontext"})
	carrier := make(map[string]string)
	prop.Inject(ctx, carrier)

	if carrier["traceparent"] == "" {
		t.Error("expected non-empty traceparent header")
	}

	newCtx := prop.Extract(context.Background(), carrier)
	extracted := SpanFromContext(newCtx)
	if extracted == nil {
		t.Fatal("expected span in extracted context")
	}
	if extracted.TraceID != span.TraceID {
		t.Errorf("expected trace ID %q, got %q", span.TraceID, extracted.TraceID)
	}
}

func TestPropagator_Formats(t *testing.T) {
	prop := NewPropagator([]string{"tracecontext", "baggage"})
	fmts := prop.Formats()
	if len(fmts) != 2 {
		t.Errorf("expected 2 formats, got %d", len(fmts))
	}
}

func TestPropagator_EmptyCarrier(t *testing.T) {
	prop := NewPropagator(nil)
	newCtx := prop.Extract(context.Background(), make(map[string]string))
	if SpanFromContext(newCtx) != nil {
		t.Error("expected nil span from empty carrier")
	}
}

// ---------------------------------------------------------------------------
// Exporter tests
// ---------------------------------------------------------------------------

func TestExporter_NewExporter(t *testing.T) {
	exp, err := NewExporter(OTLPConfig{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
	}, "test-svc", "1.0.0", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.Endpoint() != "localhost:4317" {
		t.Errorf("expected endpoint 'localhost:4317', got %q", exp.Endpoint())
	}
}

func TestExporter_EmptyEndpoint(t *testing.T) {
	_, err := NewExporter(OTLPConfig{}, "test", "1.0", "dev")
	if err == nil {
		t.Error("expected error for empty endpoint")
	}
}

func TestExporter_ExportSpans(t *testing.T) {
	exp, _ := NewExporter(OTLPConfig{Endpoint: "localhost:4317"}, "test", "1.0", "dev")

	spans := []*Span{
		{TraceID: "abc", SpanID: "123", Name: "test"},
	}

	ctx := context.Background()
	if err := exp.ExportSpans(ctx, spans); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.ExportCount() != 1 {
		t.Errorf("expected export count 1, got %d", exp.ExportCount())
	}
}

func TestExporter_ExportSpans_Empty(t *testing.T) {
	exp, _ := NewExporter(OTLPConfig{Endpoint: "localhost:4317"}, "test", "1.0", "dev")
	if err := exp.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.ExportCount() != 0 {
		t.Errorf("expected export count 0, got %d", exp.ExportCount())
	}
}

func TestExporter_ExportSpans_Cancelled(t *testing.T) {
	exp, _ := NewExporter(OTLPConfig{Endpoint: "localhost:4317"}, "test", "1.0", "dev")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := exp.ExportSpans(ctx, []*Span{{TraceID: "abc"}})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestExporter_Shutdown(t *testing.T) {
	exp, _ := NewExporter(OTLPConfig{Endpoint: "localhost:4317"}, "test", "1.0", "dev")
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Health server tests
// ---------------------------------------------------------------------------

func TestHealthServer_LivenessReadiness(t *testing.T) {
	hs := NewHealthServer(HealthConfig{Port: 8081, ReadinessPort: 8082, Enabled: true})

	if !hs.Liveness() {
		t.Error("expected healthy on creation")
	}
	if !hs.Readiness() {
		t.Error("expected ready on creation")
	}

	hs.SetHealthy(false)
	if hs.Liveness() {
		t.Error("expected unhealthy after SetHealthy(false)")
	}

	hs.SetReady(false)
	if hs.Readiness() {
		t.Error("expected not ready after SetReady(false)")
	}
}

func TestHealthServer_DependencyCheck(t *testing.T) {
	hs := NewHealthServer(HealthConfig{Port: 8081, Enabled: true})

	hs.CheckDependency("database", true)
	if !hs.Liveness() {
		t.Error("expected healthy with all dependencies up")
	}

	hs.CheckDependency("database", false)
	if hs.Liveness() {
		t.Error("expected unhealthy when dependency is down")
	}

	// Bring it back.
	hs.CheckDependency("database", true)
	if !hs.Liveness() {
		t.Error("expected healthy after dependency recovery")
	}
}

func TestHealthServer_StatusChangeCallback(t *testing.T) {
	hs := NewHealthServer(HealthConfig{Port: 8081, Enabled: true})

	var called bool
	var lastHealthy bool
	hs.OnStatusChange(func(healthy, ready bool) {
		called = true
		lastHealthy = healthy
	})

	hs.SetHealthy(false)
	if !called {
		t.Error("expected callback to be called")
	}
	if lastHealthy {
		t.Error("expected callback to report unhealthy")
	}
}

func TestHealthServer_StartShutdown(t *testing.T) {
	ctx := context.Background()
	hs := NewHealthServer(HealthConfig{Port: 9999, ReadinessPort: 9998, Enabled: true})

	if err := hs.Start(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hs.Liveness() {
		t.Error("expected healthy after start")
	}

	if err := hs.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hs.Liveness() {
		t.Error("expected unhealthy after shutdown")
	}
}

// ---------------------------------------------------------------------------
// Provider tests
// ---------------------------------------------------------------------------

func TestProvider_NewAndShutdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Health.Enabled = false // skip actual TCP listener for unit test

	provider, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !provider.IsReady() {
		t.Error("expected provider to be ready")
	}

	if provider.Tracer() == nil {
		t.Error("expected non-nil tracer")
	}
	if provider.Meter() == nil {
		t.Error("expected non-nil meter")
	}
	if provider.Exporter() == nil {
		t.Error("expected non-nil exporter")
	}
	if provider.Propagator() == nil {
		t.Error("expected non-nil propagator")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.IsReady() {
		t.Error("expected provider to not be ready after shutdown")
	}
}

func TestProvider_MetricsDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Metrics.Enabled = false
	cfg.Health.Enabled = false

	provider, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Meter() != nil {
		t.Error("expected nil meter when metrics disabled")
	}
}

func TestProvider_InvalidOTLP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OTLP.Endpoint = "" // invalid

	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Error("expected error for empty OTLP endpoint")
	}
}

func TestStartOperation_EndOperation(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})

	ctx, span := StartOperation(context.Background(), tracer, "deploy", map[string]string{"tier": "3"})
	if span == nil {
		t.Fatal("expected span from StartOperation")
	}

	EndOperation(tracer, span, nil)
	if span.Status.Code != StatusCodeOK {
		t.Errorf("expected StatusCodeOK, got %d", span.Status.Code)
	}

	// Test with error.
	_, span2 := StartOperation(context.Background(), tracer, "fail-op", nil)
	EndOperation(tracer, span2, context.DeadlineExceeded)
	if span2.Status.Code != StatusCodeError {
		t.Errorf("expected StatusCodeError, got %d", span2.Status.Code)
	}
}

func TestRecordLatency(t *testing.T) {
	tracer := NewTracer("test", SamplingConfig{Strategy: "always"})
	_, span := tracer.StartSpan(context.Background(), "op", SpanKindInternal)

	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	RecordLatency(span, "duration", start)

	if span.Attributes["duration.duration_ms"] == "" {
		t.Error("expected duration attribute to be set")
	}
}
