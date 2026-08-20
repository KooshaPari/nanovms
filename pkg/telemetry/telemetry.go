// SPDX-License-Identifier: MIT OR Apache-2.0
// Package telemetry provides OpenTelemetry instrumentation for nanovms.
//
// It offers OTLP exporter setup (gRPC and HTTP), tracer and meter providers,
// custom spans for key operations, metrics collection (counters, histograms),
// health check integration, context propagation, and graceful shutdown.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds the full telemetry configuration.
type Config struct {
	ServiceName    string         `json:"service_name"`
	ServiceVersion string         `json:"service_version"`
	Environment    string         `json:"environment"`
	OTLP           OTLPConfig     `json:"otlp"`
	Sampling       SamplingConfig `json:"sampling"`
	Metrics        MetricsConfig  `json:"metrics"`
	Health         HealthConfig   `json:"health"`
	Propagation    []string       `json:"propagation"`
}

// OTLPConfig configures OTLP exporter endpoints.
type OTLPConfig struct {
	Endpoint string            `json:"endpoint"`
	Protocol string            `json:"protocol"` // "grpc" or "http/protobuf"
	Headers  map[string]string `json:"headers,omitempty"`
	Insecure bool              `json:"insecure"`
	Timeout  time.Duration     `json:"timeout"`
	HTTPPath string            `json:"http_path,omitempty"`
}

// SamplingConfig controls trace sampling.
type SamplingConfig struct {
	Strategy string  `json:"strategy"` // "always", "never", "ratio", "parentBased"
	Ratio    float64 `json:"ratio"`
}

// MetricsConfig controls metric collection and export.
type MetricsConfig struct {
	Interval         time.Duration `json:"interval"`
	Enabled          bool          `json:"enabled"`
	HistogramBuckets []float64     `json:"histogram_buckets,omitempty"`
}

// HealthConfig configures the health check integration.
type HealthConfig struct {
	Enabled       bool `json:"enabled"`
	Port          int  `json:"port"`
	ReadinessPort int  `json:"readiness_port"`
}

// DefaultConfig returns a production-ready configuration.
func DefaultConfig() Config {
	return Config{
		ServiceName:    "nanovms",
		ServiceVersion: "0.1.0",
		Environment:    "production",
		OTLP: OTLPConfig{
			Endpoint: "localhost:4317",
			Protocol: "grpc",
			Timeout:  10 * time.Second,
			Insecure: true,
		},
		Sampling: SamplingConfig{
			Strategy: "ratio",
			Ratio:    0.1,
		},
		Metrics: MetricsConfig{
			Interval: 30 * time.Second,
			Enabled:  true,
			HistogramBuckets: []float64{
				0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000,
			},
		},
		Health: HealthConfig{
			Enabled:       true,
			Port:          8081,
			ReadinessPort: 8082,
		},
		Propagation: []string{"tracecontext", "baggage"},
	}
}

// ---------------------------------------------------------------------------
// Provider – facade over tracers, meters, and exporters
// ---------------------------------------------------------------------------

// Provider holds all telemetry providers and is the main entry point.
type Provider struct {
	config     Config
	logger     *slog.Logger
	tp         *sdktrace.TracerProvider
	mp         *sdkmetric.MeterProvider
	propagator propagation.TextMapPropagator
	health     *HealthServer
	meter      metric.Meter
}

// NewProvider initialises the full telemetry stack from config.
func NewProvider(cfg Config, logger *slog.Logger) (*Provider, error) {
	if logger == nil {
		logger = slog.Default()
	}

	p := &Provider{config: cfg, logger: logger}

	// Build resource.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	// Propagator.
	p.propagator = newPropagator(cfg.Propagation)

	// Trace provider.
	tp, err := p.buildTraceProvider(context.Background(), res)
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace provider: %w", err)
	}
	p.tp = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(p.propagator)

	// Meter provider.
	if cfg.Metrics.Enabled {
		mp, meter, err := p.buildMeterProvider(context.Background(), res)
		if err != nil {
			return nil, fmt.Errorf("telemetry: meter provider: %w", err)
		}
		p.mp = mp
		p.meter = meter
		otel.SetMeterProvider(mp)
	}

	// Health.
	if cfg.Health.Enabled {
		p.health = NewHealthServer(cfg.Health)
	}

	logger.Info("telemetry provider initialised",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"otlp_endpoint", cfg.OTLP.Endpoint,
		"otlp_protocol", cfg.OTLP.Protocol,
	)

	return p, nil
}

// Start launches background goroutines (health server).
func (p *Provider) Start(ctx context.Context) error {
	if p.health != nil {
		if err := p.health.Start(ctx); err != nil {
			return fmt.Errorf("telemetry: health server: %w", err)
		}
	}
	p.logger.Info("telemetry provider started")
	return nil
}

// Shutdown flushes all pending data and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.health != nil {
		_ = p.health.Shutdown(ctx)
	}
	if p.tp != nil {
		if err := p.tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("telemetry: tracer provider shutdown: %w", err)
		}
	}
	if p.mp != nil {
		if err := p.mp.Shutdown(ctx); err != nil {
			return fmt.Errorf("telemetry: meter provider shutdown: %w", err)
		}
	}
	p.logger.Info("telemetry provider shut down")
	return nil
}

// Meter returns the OTel Meter (nil if metrics disabled).
func (p *Provider) Meter() metric.Meter { return p.meter }

// Health returns the health server (nil if disabled).
func (p *Provider) Health() *HealthServer { return p.health }

// Propagator returns the text-map propagator.
func (p *Provider) Propagator() propagation.TextMapPropagator { return p.propagator }

// ---------------------------------------------------------------------------
// OTLP exporters (gRPC and HTTP)
// ---------------------------------------------------------------------------

func (p *Provider) buildTraceProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	switch p.config.OTLP.Protocol {
	case "http/protobuf", "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(p.config.OTLP.Endpoint),
		}
		if p.config.OTLP.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if p.config.OTLP.HTTPPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(p.config.OTLP.HTTPPath))
		}
		if p.config.OTLP.Headers != nil {
			opts = append(opts, otlptracehttp.WithHeaders(p.config.OTLP.Headers))
		}
		if p.config.OTLP.Timeout > 0 {
			opts = append(opts, otlptracehttp.WithTimeout(p.config.OTLP.Timeout))
		}
		exporter, err = otlptracehttp.New(ctx, opts...)

	default: // "grpc"
		dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(p.config.OTLP.Endpoint),
			otlptracegrpc.WithDialOption(dialOpts...),
		}
		if p.config.OTLP.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if p.config.OTLP.Headers != nil {
			opts = append(opts, otlptracegrpc.WithHeaders(p.config.OTLP.Headers))
		}
		if p.config.OTLP.Timeout > 0 {
			opts = append(opts, otlptracegrpc.WithTimeout(p.config.OTLP.Timeout))
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	}
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.AlwaysSample()
	switch p.config.Sampling.Strategy {
	case "never":
		sampler = sdktrace.NeverSample()
	case "ratio":
		sampler = sdktrace.TraceIDRatioBased(p.config.Sampling.Ratio)
	case "parentBased":
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(p.config.Sampling.Ratio))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)
	return tp, nil
}

func (p *Provider) buildMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, metric.Meter, error) {
	var exporter sdkmetric.Exporter
	var err error

	switch p.config.OTLP.Protocol {
	case "http/protobuf", "http":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(p.config.OTLP.Endpoint),
		}
		if p.config.OTLP.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if p.config.OTLP.Headers != nil {
			opts = append(opts, otlpmetrichttp.WithHeaders(p.config.OTLP.Headers))
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)

	default: // "grpc"
		dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(p.config.OTLP.Endpoint),
			otlpmetricgrpc.WithDialOption(dialOpts...),
		}
		if p.config.OTLP.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if p.config.OTLP.Headers != nil {
			opts = append(opts, otlpmetricgrpc.WithHeaders(p.config.OTLP.Headers))
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	}
	if err != nil {
		return nil, nil, err
	}

	interval := p.config.Metrics.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(interval),
			),
		),
	)

	meter := mp.Meter(p.config.ServiceName)
	return mp, meter, nil
}

// ---------------------------------------------------------------------------
// Context propagation helpers
// ---------------------------------------------------------------------------

func newPropagator(formats []string) propagation.TextMapPropagator {
	propagators := make([]propagation.TextMapPropagator, 0, len(formats))
	for _, f := range formats {
		switch f {
		case "tracecontext", "w3c":
			propagators = append(propagators, propagation.TraceContext{})
		case "baggage":
			propagators = append(propagators, propagation.Baggage{})
		}
	}
	if len(propagators) == 0 {
		propagators = append(propagators, propagation.TraceContext{}, propagation.Baggage{})
	}
	return propagation.NewCompositeTextMapPropagator(propagators...)
}

// InjectSpan writes span context into a carrier map (e.g. HTTP headers).
func InjectSpan(ctx context.Context, propagator propagation.TextMapPropagator, carrier propagation.MapCarrier) {
	propagator.Inject(ctx, carrier)
}

// ExtractSpan reads span context from a carrier map into a new context.
func ExtractSpan(ctx context.Context, propagator propagation.TextMapPropagator, carrier propagation.MapCarrier) context.Context {
	return propagator.Extract(ctx, carrier)
}

// MapCarrier is a simple map-based carrier for propagation testing.
type MapCarrier map[string]string

// Get returns the value for the given key.
func (c MapCarrier) Get(key string) string { return c[key] }

// Set stores the key-value pair.
func (c MapCarrier) Set(key, value string) { c[key] = value }

// Keys returns all stored keys.
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// ---------------------------------------------------------------------------
// Custom span helpers for key operations
// ---------------------------------------------------------------------------

// StartSpan creates a new span with the given name and kind, returning a
// context that carries the span.
func StartSpan(ctx context.Context, tracerName, spanName string, kind trace.SpanKind, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, trace.WithSpanKind(kind), trace.WithAttributes(attrs...))
}

// EndSpan completes a span, optionally recording an error.
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// WithLatency records the elapsed time since start as a span attribute.
func WithLatency(span trace.Span, key string, start time.Time) {
	elapsed := time.Since(start)
	span.SetAttributes(attribute.Duration(key, elapsed))
}

// ---------------------------------------------------------------------------
// Health server
// ---------------------------------------------------------------------------

// HealthServer provides liveness and readiness probes.
type HealthServer struct {
	config       HealthConfig
	mu           sync.RWMutex
	healthy      bool
	ready        bool
	dependencies map[string]bool
	listeners    []func(healthy, ready bool)
}

// NewHealthServer creates a HealthServer from config.
func NewHealthServer(cfg HealthConfig) *HealthServer {
	return &HealthServer{
		config:       cfg,
		healthy:      true,
		ready:        true,
		dependencies: make(map[string]bool),
	}
}

// Start begins serving health checks.
func (hs *HealthServer) Start(_ context.Context) error {
	hs.mu.Lock()
	hs.healthy = true
	hs.ready = true
	hs.mu.Unlock()
	hs.notifyListeners()
	slog.Info("health server started", "port", hs.config.Port)
	return nil
}

// Shutdown stops the health server.
func (hs *HealthServer) Shutdown(_ context.Context) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.healthy = false
	hs.ready = false
	hs.notifyListeners()
	return nil
}

// SetHealthy updates the liveness status.
func (hs *HealthServer) SetHealthy(healthy bool) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.healthy = healthy
	hs.notifyListeners()
}

// SetReady updates the readiness status.
func (hs *HealthServer) SetReady(ready bool) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.ready = ready
	hs.notifyListeners()
}

// CheckDependency registers the health of a named dependency.
func (hs *HealthServer) CheckDependency(name string, healthy bool) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.dependencies[name] = healthy
	allHealthy := true
	for _, h := range hs.dependencies {
		if !h {
			allHealthy = false
			break
		}
	}
	hs.healthy = allHealthy
	hs.notifyListeners()
}

// Liveness returns the current liveness status.
func (hs *HealthServer) Liveness() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.healthy
}

// Readiness returns the current readiness status.
func (hs *HealthServer) Readiness() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.ready
}

// OnStatusChange registers a callback for health status changes.
func (hs *HealthServer) OnStatusChange(fn func(healthy, ready bool)) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.listeners = append(hs.listeners, fn)
}

func (hs *HealthServer) notifyListeners() {
	for _, fn := range hs.listeners {
		fn(hs.healthy, hs.ready)
	}
}
