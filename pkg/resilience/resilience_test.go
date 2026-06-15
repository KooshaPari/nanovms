// SPDX-License-Identifier: MIT OR Apache-2.0
package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryPolicy_ExecutesFn(t *testing.T) {
	var calls int
	fn := func() error {
		calls++
		return nil
	}

	p := &RetryPolicy{MaxAttempts: 3}
	if err := p.Execute(context.Background(), fn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryPolicy_RetriesAndSucceeds(t *testing.T) {
	var calls int
	fn := func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}

	p := &RetryPolicy{
		MaxAttempts: 5,
		BackoffFunc: func(attempt int) time.Duration { return 1 * time.Millisecond },
	}
	if err := p.Execute(context.Background(), fn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetryPolicy_RetriesExhausted(t *testing.T) {
	fn := func() error {
		return errors.New("always fails")
	}

	p := &RetryPolicy{
		MaxAttempts: 3,
		BackoffFunc: func(attempt int) time.Duration { return 1 * time.Millisecond },
	}
	if err := p.Execute(context.Background(), fn); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRetryPolicy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &RetryPolicy{MaxAttempts: 3}
	if err := p.Execute(ctx, func() error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 5*time.Minute)

	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Fatal("expected Allow to be true before threshold")
		}
		cb.RecordFailure()
	}

	if cb.Allow() {
		t.Fatal("expected Allow to be false after threshold")
	}
}

func TestCircuitBreaker_HalfOpenAllowsProbe(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	// Trip the breaker
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.Allow() {
		t.Fatal("expected Allow to be false while open")
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("expected Allow to be true in half-open state")
	}
}

func TestCircuitBreaker_HalfOpenClosesOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("expected half-open probe to be allowed")
	}
	cb.RecordSuccess()

	if !cb.Allow() {
		t.Fatal("expected closed state after half-open success")
	}
}

func TestCircuitBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 50*time.Millisecond)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("expected half-open probe to be allowed")
	}
	cb.RecordFailure()

	if cb.Allow() {
		t.Fatal("expected open state after half-open failure")
	}
}

func TestCircuitBreaker_ClosedRecordsSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 1, 5*time.Minute)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.state != StateClosed {
		t.Fatalf("expected closed state, got %d", cb.state)
	}

	cb.Allow()
	cb.RecordSuccess()

	if cb.failureCount != 0 {
		t.Fatalf("expected failureCount reset, got %d", cb.failureCount)
	}

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.state != StateOpen {
		t.Fatalf("expected open state, got %d", cb.state)
	}
}
