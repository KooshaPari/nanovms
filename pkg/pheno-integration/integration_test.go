// SPDX-License-Identifier: MIT OR Apache-2.0
package phenointegration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// uuidV4Pattern matches the canonical 8-4-4-4-12 hex form and verifies
// the version (4) and variant (8/9/a/b) nibbles.
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestInitServerMiddlewareInjectsRequestID asserts that a request
// through the handler returned by InitServer carries a valid X-Request-ID
// header in the response.
func TestInitServerMiddlewareInjectsRequestID(t *testing.T) {
	ctx := context.Background()
	handler := InitServer(ctx)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	id := rr.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("X-Request-ID header missing from response")
	}
	if !uuidV4Pattern.MatchString(id) {
		t.Fatalf("X-Request-ID %q is not a valid UUID v4", id)
	}
}

// TestHealthzReturns200 asserts that the /healthz endpoint returns
// HTTP 200.
func TestHealthzReturns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	HandleHealthz(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("HandleHealthz status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestHealthEndpointReturns200 asserts that the /health endpoint returns
// HTTP 200, matching the documented API surface (audit report L5/L27).
func TestHealthEndpointReturns200(t *testing.T) {
	ctx := context.Background()
	handler := InitServer(ctx)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestMetricsEndpointReturns200 asserts that the /metrics endpoint
// returns HTTP 200 with the correct content type.
func TestMetricsEndpointReturns200(t *testing.T) {
	ctx := context.Background()
	handler := InitServer(ctx)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want %d", rr.Code, http.StatusOK)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "nanovms_http_requests_total") || !strings.Contains(body, "nanovms_uptime_seconds") {
		t.Fatalf("/metrics body missing expected metrics:\n%s", body)
	}
}

// TestMetricsCountsRequests asserts that requests served through InitServer
// are counted in the /metrics output.
func TestMetricsCountsRequests(t *testing.T) {
	// Reset metrics for clean test
	globalMetrics = newMetricsCollector()
	ctx := context.Background()
	handler := InitServer(ctx)

	// Make a request through the handler
	req1 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Check metrics
	reqM := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, reqM)
	body := rr.Body.String()
	if !strings.Contains(body, "nanovms_http_requests_total 2") {
		t.Fatalf("expected 2 requests counted (healthz + metrics read), got:\n%s", body)
	}
}
