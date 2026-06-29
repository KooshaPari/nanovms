// SPDX-License-Identifier: MIT OR Apache-2.0
package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestRateLimiterAllowsInitialBurst verifies that the first burst
// requests are all allowed.
func TestRateLimiterAllowsInitialBurst(t *testing.T) {
	rl := New(100, 10)
	for i := 0; i < 10; i++ {
		if !rl.Allow() {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
}

// TestRateLimiterBlocksAfterBurst verifies that the limiter blocks
// once the burst is consumed and no time has elapsed.
func TestRateLimiterBlocksAfterBurst(t *testing.T) {
	rl := New(100, 5)
	for i := 0; i < 5; i++ {
		rl.Allow()
	}
	if rl.Allow() {
		t.Fatal("request should be blocked after burst exhausted")
	}
}

// TestRateLimiterRefills verifies that tokens are refilled over time.
func TestRateLimiterRefills(t *testing.T) {
	// 10 tokens/sec, burst 1 → after 100ms we get 1 token back
	rl := New(10, 1)
	if !rl.Allow() {
		t.Fatal("initial request should be allowed")
	}
	if rl.Allow() {
		t.Fatal("second request should be blocked (burst=1)")
	}

	time.Sleep(150 * time.Millisecond) // enough for ~1.5 tokens

	if !rl.Allow() {
		t.Fatal("request should be allowed after refill")
	}
}

// TestRateLimiterPassThrough verifies that rate <= 0 allows everything.
func TestRateLimiterPassThrough(t *testing.T) {
	rl := New(0, 1)
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Fatal("rate <= 0 should allow all requests")
		}
	}
}

// TestRateLimiterConcurrentSafety verifies that Allow() is safe under
// concurrent access.
func TestRateLimiterConcurrentSafety(t *testing.T) {
	rl := New(1000, 100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				rl.Allow()
			}
		}()
	}
	wg.Wait()
}

// TestMiddlewareReturns429 verifies that the HTTP middleware returns
// 429 when the limit is exceeded.
func TestMiddlewareReturns429(t *testing.T) {
	rl := New(100, 1)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request passes
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: got status %d, want %d", rr.Code, http.StatusOK)
	}

	// Second request blocked (burst=1, no refill)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: got status %d, want %d", rr2.Code, http.StatusTooManyRequests)
	}
}

// TestMiddlewarePassesThrough verifies the middleware passes through
// requests when rate limit is not exceeded.
func TestMiddlewarePassesThrough(t *testing.T) {
	rl := New(0, 1) // pass-through
	var called bool
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rr.Code, http.StatusOK)
	}
}
