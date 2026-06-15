// SPDX-License-Identifier: MIT OR Apache-2.0
// Package resilience provides hand-rolled retry and circuit-breaker primitives
// using only the Go standard library.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RetryPolicy defines how to retry an operation with backoff.
type RetryPolicy struct {
	MaxAttempts int
	BackoffFunc func(attempt int) time.Duration
}

// Execute runs fn up to MaxAttempts times, returning the last error if all
// attempts fail. A zero or negative MaxAttempts defaults to 1 attempt.
func (r *RetryPolicy) Execute(ctx context.Context, fn func() error) error {
	attempts := r.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if i < attempts-1 && r.BackoffFunc != nil {
			delay := r.BackoffFunc(i + 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// State represents the current state of a CircuitBreaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker prevents cascading failures by rejecting requests when a
// failure threshold is exceeded.
type CircuitBreaker struct {
	mu sync.Mutex

	state           State
	failureCount    int
	failureThreshold int
	successCount    int
	successThreshold int
	resetTimeout    time.Duration
	lastFailureTime time.Time
}

// NewCircuitBreaker creates a CircuitBreaker with sensible defaults.
// A failureThreshold of 0 disables the circuit breaker (always allows).
func NewCircuitBreaker(failureThreshold, successThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		resetTimeout:     resetTimeout,
	}
}

// Allow returns true if the request should be permitted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failureThreshold <= 0 {
		return true
	}

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
			cb.successCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess should be called when a permitted request succeeds.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failureThreshold <= 0 {
		return
	}

	cb.failureCount = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.successCount = 0
		}
	case StateClosed:
		// nothing to do
	}
}

// RecordFailure should be called when a permitted request fails.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failureThreshold <= 0 {
		return
	}

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.failureCount++
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open and a request
// is not allowed.
var ErrCircuitOpen = errors.New("circuit breaker is open")
