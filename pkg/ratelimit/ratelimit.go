// SPDX-License-Identifier: MIT OR Apache-2.0
//
// Package ratelimit provides a token-bucket rate limiter suitable for
// HTTP middleware.  It uses only the Go standard library and is safe for
// concurrent use.
//
// Usage:
//
//	limiter := ratelimit.New(100, 100)  // 100 requests/sec, burst 100
//	mux := http.NewServeMux()
//	mux.Handle("/api/", limiter.Middleware(apiHandler))
//
// The audit report (L25) flagged that nanoVMS documents a rate_limit
// configuration value (1000 req/min in docs/reference/configuration.md)
// but had no enforcement code.  This package fills that gap.
package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token-bucket rate limiter.
//
// The bucket is refilled at `rate` tokens per second up to a maximum
// of `burst` tokens.  Each call to Allow() consumes one token.
// When the bucket is empty, Allow() returns false and the caller
// should reject the request (typically with HTTP 429).
type RateLimiter struct {
	mu    sync.Mutex
	rate  float64
	burst int

	tokens   float64
	lastTime time.Time
}

// New creates a RateLimiter that allows `rate` requests per second
// with a maximum burst of `burst`.  If rate <= 0 the limiter allows
// all requests (pass-through).  burst must be at least 1.
func New(rate float64, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow consumes one token and returns true if the request is within
// the rate limit.  Returns false when the bucket is exhausted.
func (rl *RateLimiter) Allow() bool {
	if rl.rate <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.lastTime = now

	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// Middleware returns an http.Handler that rate-limits incoming requests.
// When the rate limit is exceeded it responds with HTTP 429
// (Too Many Requests) and a plain-text message.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow() {
			http.Error(w, "429 Too Many Requests\n", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
