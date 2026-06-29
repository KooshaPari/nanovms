// SPDX-License-Identifier: MIT OR Apache-2.0
//
// Package phenointegration is a thin HTTP middleware helper used by the
// nanovms daemon and any future pheno-internal services to attach a
// per-request X-Request-Id header (echoing inbound, generating otherwise)
// and structured logging via the standard `log/slog` package.
//
// Historically this package depended on `github.com/kooshapari/pheno-go-ctxkit/ctxkit`,
// but that dependency pulled a non-trivial graph (and required a
// `replace` directive to a worktree path that only existed on the original
// author's machine). The ctxkit pattern is small enough to inline
// directly here, which makes `nanovms` self-contained and reproducible to
// build from a clean clone. The package remains import-compatible: any
// caller of `InitServer(ctx)` continues to get the same http.Handler
// shape.
package phenointegration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// requestIDHeader is the HTTP header used to carry the per-request
// identifier end-to-end.
const requestIDHeader = "X-Request-Id"

// InitServer returns an http.Handler with the request-id middleware
// applied. The returned handler registers /healthz and can be passed
// directly to http.Server or wrapped by additional middleware.
//
// The middleware is intentionally tiny: it generates a UUID-like ID per
// request, threads it through the request context (so downstream
// handlers can read it via RequestIDFrom), echoes it on the response,
// and emits a single structured log line at request completion.
// InitServer returns an http.Handler with the request-id middleware
// applied. The returned handler registers /healthz, /health (alias),
// and /metrics endpoints and can be passed directly to http.Server or
// wrapped by additional middleware.
func InitServer(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HandleHealthz)
	mux.HandleFunc("/health", HandleHealth)
	mux.HandleFunc("/metrics", HandleMetrics)
	return requestIDMiddleware(mux)
}

// HandleHealthz is a simple liveness probe that returns HTTP 200.
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HandleHealth is an alias for /healthz — it returns HTTP 200 as a
// liveness probe.  The audit report (L5, L27) flagged that the API
// reference documents /health but only /healthz was implemented.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// globalMetrics is a package-level collector for the /metrics endpoint.
var globalMetrics = newMetricsCollector()

// metricsCollector holds cumulative counters exposed at /metrics.
type metricsCollector struct {
	mu        sync.Mutex
	requests  int64
	startTime time.Time
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{startTime: time.Now()}
}

// recordRequest increments the request counter.
func (mc *metricsCollector) recordRequest() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.requests++
}

// snapshot returns a copy of the current counters.
func (mc *metricsCollector) snapshot() (requests int64, uptime time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.requests, time.Since(mc.startTime)
}

// HandleMetrics serves basic runtime metrics in Prometheus
// exposition format.  The audit report (L5, L27) flagged that the API
// reference documents /metrics but no handler existed in code.
func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	reqs, uptime := globalMetrics.snapshot()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "# HELP nanovms_http_requests_total Total HTTP requests processed.\n")
	fmt.Fprintf(w, "# TYPE nanovms_http_requests_total counter\n")
	fmt.Fprintf(w, "nanovms_http_requests_total %d\n", reqs)
	fmt.Fprintf(w, "# HELP nanovms_uptime_seconds Server uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE nanovms_uptime_seconds gauge\n")
	fmt.Fprintf(w, "nanovms_uptime_seconds %.0f\n", uptime.Seconds())
}

// contextKey is unexported to prevent key collisions in the request
// context.
type contextKey string

const requestIDKey contextKey = "requestID"

// RequestIDFrom returns the per-request identifier previously attached by
// requestIDMiddleware. Returns the empty string if no ID is present.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// requestIDMiddleware wraps an http.Handler so every request gets a
// stable X-Request-Id header. Inbound requests that already carry the
// header have it preserved; otherwise a 16-byte hex ID is generated.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		globalMetrics.recordRequest()
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		slog.Info("http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", id),
		)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
