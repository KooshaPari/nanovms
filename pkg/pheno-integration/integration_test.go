// SPDX-License-Identifier: MIT OR Apache-2.0
package phenointegration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
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

// TestInitServerHealthzEndpointReturns200 asserts that the /healthz
// endpoint served by the handler returned by InitServer returns HTTP 200.
func TestInitServerHealthzEndpointReturns200(t *testing.T) {
	ctx := context.Background()
	handler := InitServer(ctx)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want %d", rr.Code, http.StatusOK)
	}
}
