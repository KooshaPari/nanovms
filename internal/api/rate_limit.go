package api

import (
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements a simple token-bucket rate limiter.
// Refill: rate tokens per second, max burst = burst.
type TokenBucket struct {
	mu      sync.Mutex
	rate    float64
	burst   int
	tokens  float64
	lastRef time.Time
}

// NewTokenBucket creates a token bucket with the given rate (tokens/s) and burst capacity.
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:    rate,
		burst:   burst,
		tokens:  float64(burst),
		lastRef: time.Now(),
	}
}

// Allow returns true if a token can be consumed.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}
	return false
}

// Remaining returns the current token count (approximate).
func (tb *TokenBucket) Remaining() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRef).Seconds()
	tb.lastRef = now
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
}

// RateLimitMiddleware returns an HTTP middleware that enforces a global
// 100 req/s rate limit using a token bucket. On exhaustion it returns
// 429 with a Retry-After header.
func RateLimitMiddleware(next http.Handler) http.Handler {
	tb := NewTokenBucket(100, 100)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !tb.Allow() {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate_limit_exceeded","message":"try again in 1 second"}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
