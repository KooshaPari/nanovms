package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitMiddlewareAllowsBelowThreshold(t *testing.T) {
	handler := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitMiddlewareReturns429AfterExhaustion(t *testing.T) {
	handler := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the token bucket (burst = 100).
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected all initial 100 to pass; request %d got %d", i, rec.Code)
		}
	}

	// Next request should be rate-limited.
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exhaustion, got %d", rec.Code)
	}

	if retry := rec.Header().Get("Retry-After"); retry != "1" {
		t.Fatalf("expected Retry-After: 1 header, got %q", retry)
	}
}

func TestRateLimitMiddlewareReturnsJSONError(t *testing.T) {
	handler := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust bucket.
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/sandbox", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty JSON error body")
	}
}
