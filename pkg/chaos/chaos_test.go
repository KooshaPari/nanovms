// SPDX-License-Identifier: MIT OR Apache-2.0
package chaos

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ── NetworkPartition Tests ───────────────────────

func TestNetworkPartition_ApplyRevert(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, []string{"db.internal"})
	if err := np.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !np.IsApplied() {
		t.Fatal("expected partition to be applied")
	}
	if err := np.Apply(); err != ErrAlreadyApplied {
		t.Fatalf("expected ErrAlreadyApplied, got %v", err)
	}

	if err := np.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if np.IsApplied() {
		t.Fatal("expected partition to be reverted")
	}
	if err := np.Revert(); err != ErrNotApplied {
		t.Fatalf("expected ErrNotApplied, got %v", err)
	}
}

func TestNetworkPartition_IsBlocked(t *testing.T) {
	np := NewNetworkPartition(
		[]string{"10.0.0.1", "10.0.0.2"},
		[]string{"db.internal", "cache.internal"},
	)
	np.Apply()
	defer np.Revert()

	if !np.IsBlocked("10.0.0.1") {
		t.Fatal("expected 10.0.0.1 to be blocked")
	}
	if !np.IsBlocked("db.internal") {
		t.Fatal("expected db.internal to be blocked")
	}
	if np.IsBlocked("10.0.0.99") {
		t.Fatal("10.0.0.99 should not be blocked")
	}
}

func TestNetworkPartition_NotBlockedWhenNotApplied(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	if np.IsBlocked("10.0.0.1") {
		t.Fatal("should not be blocked when not applied")
	}
}

func TestNetworkPartition_Duration(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	np.Apply()
	time.Sleep(10 * time.Millisecond)
	np.Revert()

	d := np.Duration()
	if d < 5*time.Millisecond {
		t.Fatalf("expected duration >= 5ms, got %v", d)
	}
}

// ── ResourceExhaustion Tests ─────────────────────

func TestResourceExhaustion_DiskFull(t *testing.T) {
	re := NewResourceExhaustion("disk-full", 100, 0)
	if err := re.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !re.IsApplied() {
		t.Fatal("expected resource exhaustion to be applied")
	}

	if err := re.ConsumeBytes(50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if re.RemainingBudget() != 50 {
		t.Fatalf("expected 50 remaining, got %d", re.RemainingBudget())
	}

	if err := re.ConsumeBytes(60); err == nil {
		t.Fatal("expected error when exceeding budget")
	}

	if err := re.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
}

func TestResourceExhaustion_OOM(t *testing.T) {
	re := NewResourceExhaustion("oom", 1024*1024, 0)
	re.Apply()
	defer re.Revert()

	for i := 0; i < 10; i++ {
		if err := re.ConsumeBytes(100000); err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
	}

	if err := re.ConsumeBytes(100000); err == nil {
		t.Fatal("expected OOM budget error")
	}
}

func TestResourceExhaustion_FDExhaustion(t *testing.T) {
	re := NewResourceExhaustion("fd-exhaustion", 0, 3)
	re.Apply()
	defer re.Revert()

	for i := 0; i < 3; i++ {
		if err := re.AllocateFD(); err != nil {
			t.Fatalf("unexpected error at fd %d: %v", i, err)
		}
	}

	if err := re.AllocateFD(); err == nil {
		t.Fatal("expected fd limit error")
	}

	re.FreeFD()
	if err := re.AllocateFD(); err != nil {
		t.Fatalf("expected success after freeing fd: %v", err)
	}
}

func TestResourceExhaustion_RevertClearsState(t *testing.T) {
	re := NewResourceExhaustion("disk-full", 100, 3)
	re.Apply()
	re.ConsumeBytes(50)
	re.AllocateFD()

	re.Revert()
	re.Apply()

	if re.RemainingBudget() != 100 {
		t.Fatalf("expected budget to be reset after revert, got %d", re.RemainingBudget())
	}
}

func TestResourceExhaustion_ConcurrentAccess(t *testing.T) {
	re := NewResourceExhaustion("disk-full", 10000, 0)
	re.Apply()
	defer re.Revert()

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if err := re.ConsumeBytes(1); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	errCount := 0
	for range errs {
		errCount++
	}
	t.Logf("Got %d budget exhaustion errors (expected some)", errCount)
}

// ── ProcessKill Tests ────────────────────────────

func TestProcessKill_ApplyRevert(t *testing.T) {
	pk := NewProcessKill(12345, 100*time.Millisecond)
	if err := pk.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if err := pk.Apply(); err != ErrAlreadyApplied {
		t.Fatalf("expected ErrAlreadyApplied, got %v", err)
	}

	if err := pk.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if pk.IsKilled() {
		t.Fatal("process should not be killed after revert")
	}
}

func TestProcessKill_FiresAfterDelay(t *testing.T) {
	pk := NewProcessKill(999, 20*time.Millisecond)
	pk.Apply()

	if pk.IsKilled() {
		t.Fatal("process should not be killed immediately")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := pk.Wait(ctx); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if !pk.IsKilled() {
		t.Fatal("process should be killed after delay")
	}

	pk.Revert()
}

func TestProcessKill_CancelBeforeFire(t *testing.T) {
	pk := NewProcessKill(999, 5*time.Second)
	pk.Apply()

	time.Sleep(10 * time.Millisecond)
	pk.Revert()

	if pk.IsKilled() {
		t.Fatal("process should not be killed after revert")
	}
}

// ── LatencyInjection Tests ───────────────────────

func TestLatencyInjection_FixedDelay(t *testing.T) {
	li := NewLatencyInjection(20 * time.Millisecond)
	li.Apply()
	defer li.Revert()

	start := time.Now()
	li.Wait()
	elapsed := time.Since(start)

	if elapsed < 15*time.Millisecond {
		t.Fatalf("expected >= 15ms latency, got %v", elapsed)
	}
}

func TestLatencyInjection_RangeDelay(t *testing.T) {
	li := NewRangeLatencyInjection(10*time.Millisecond, 30*time.Millisecond)
	li.Apply()
	defer li.Revert()

	var total time.Duration
	for i := 0; i < 10; i++ {
		start := time.Now()
		li.Wait()
		total += time.Since(start)
	}

	avg := total / 10
	if avg < 5*time.Millisecond {
		t.Fatalf("expected average latency >= 5ms, got %v", avg)
	}
}

func TestLatencyInjection_NoOpWhenNotApplied(t *testing.T) {
	li := NewLatencyInjection(1 * time.Second)
	start := time.Now()
	li.Wait()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("LatencyInjection should be no-op when not applied, took %v", elapsed)
	}
}

func TestLatencyInjection_Duration(t *testing.T) {
	li := NewLatencyInjection(1 * time.Millisecond)
	li.Apply()
	time.Sleep(15 * time.Millisecond)
	li.Revert()

	d := li.Duration()
	if d < 10*time.Millisecond {
		t.Fatalf("expected duration >= 10ms, got %v", d)
	}
}

// ── ErrorInjection Tests ─────────────────────────

func TestErrorInjection_ApplyRevert(t *testing.T) {
	ei := NewErrorInjection(0.5, "503")
	if err := ei.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !ei.IsApplied() {
		t.Fatal("expected error injection to be applied")
	}
	if err := ei.Apply(); err != ErrAlreadyApplied {
		t.Fatalf("expected ErrAlreadyApplied, got %v", err)
	}

	if err := ei.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if ei.IsApplied() {
		t.Fatal("expected error injection to be reverted")
	}
}

func TestErrorInjection_ShouldFail(t *testing.T) {
	ei := NewErrorInjection(1.0, "timeout") // 100% failure rate
	ei.Apply()
	defer ei.Revert()

	for i := 0; i < 100; i++ {
		if !ei.ShouldFail() {
			t.Fatalf("expected failure at iteration %d with 100%% rate", i)
		}
	}
}

func TestErrorInjection_NoFailWhenNotApplied(t *testing.T) {
	ei := NewErrorInjection(1.0, "503")
	for i := 0; i < 10; i++ {
		if ei.ShouldFail() {
			t.Fatal("ShouldFail should return false when not applied")
		}
	}
}

func TestErrorInjection_ZeroRateNeverFails(t *testing.T) {
	ei := NewErrorInjection(0.0, "503")
	ei.Apply()
	defer ei.Revert()

	for i := 0; i < 100; i++ {
		if ei.ShouldFail() {
			t.Fatalf("should never fail at 0%% rate, failed at iteration %d", i)
		}
	}
}

func TestErrorInjection_Stats(t *testing.T) {
	ei := NewErrorInjection(0.5, "503")
	ei.Apply()
	defer ei.Revert()

	for i := 0; i < 1000; i++ {
		ei.ShouldFail()
	}

	fails, successes := ei.Stats()
	if fails+successes != 1000 {
		t.Fatalf("expected 1000 total, got %d", fails+successes)
	}
	if fails == 0 {
		t.Fatal("expected some failures with 50%% rate")
	}
}

func TestErrorInjection_ErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
	}{
		{"timeout", "timeout"},
		{"connection_refused", "connection_refused"},
		{"503", "503"},
		{"500", "500"},
		{"custom", "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ei := NewErrorInjection(1.0, tc.errorType)
			if ei.Error() == nil {
				t.Fatal("Error() should never return nil")
			}
		})
	}
}

func TestErrorInjection_Name(t *testing.T) {
	ei := NewErrorInjection(0.3, "timeout")
	name := ei.Name()
	if name == "" {
		t.Fatal("Name() should not be empty")
	}
}

// ── DiskFill Tests ───────────────────────────────

func TestDiskFill_ApplyRevert(t *testing.T) {
	df := NewDiskFill("/tmp/test", 1024*1024)
	if err := df.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !df.IsApplied() {
		t.Fatal("expected disk fill to be applied")
	}

	if err := df.Write(512); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if df.FilledBytes() != 512 {
		t.Fatalf("expected 512 filled bytes, got %d", df.FilledBytes())
	}

	if err := df.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if df.FilledBytes() != 0 {
		t.Fatal("expected 0 filled bytes after revert")
	}
}

func TestDiskFill_Exhausted(t *testing.T) {
	df := NewDiskFill("/tmp/test", 100)
	df.Apply()
	defer df.Revert()

	if err := df.Write(80); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := df.Write(30); err == nil {
		t.Fatal("expected error when disk is full")
	}
}

func TestDiskFill_NoOpWhenNotApplied(t *testing.T) {
	df := NewDiskFill("/tmp/test", 100)
	if err := df.Write(200); err != nil {
		t.Fatalf("Write should be no-op when not applied: %v", err)
	}
	if df.FilledBytes() != 0 {
		t.Fatal("expected 0 filled bytes when not applied")
	}
}

func TestDiskFill_DoubleApply(t *testing.T) {
	df := NewDiskFill("/tmp/test", 100)
	df.Apply()
	if err := df.Apply(); err != ErrAlreadyApplied {
		t.Fatalf("expected ErrAlreadyApplied, got %v", err)
	}
	df.Revert()
}

func TestDiskFill_DoubleRevert(t *testing.T) {
	df := NewDiskFill("/tmp/test", 100)
	df.Apply()
	df.Revert()
	if err := df.Revert(); err != ErrNotApplied {
		t.Fatalf("expected ErrNotApplied, got %v", err)
	}
}

func TestDiskFill_Name(t *testing.T) {
	df := NewDiskFill("/data", 4096)
	if df.Name() == "" {
		t.Fatal("Name() should not be empty")
	}
}

// ── CircuitBreaker Tests ─────────────────────────

func TestCircuitBreaker_NormalOperations(t *testing.T) {
	cb := NewCircuitBreaker(3, 2)
	cb.Apply()
	defer cb.Revert()

	if !cb.Allow() {
		t.Fatal("circuit breaker should allow in closed state")
	}

	cb.RecordFailure()
	if cb.GetState() != StateClosed {
		t.Fatalf("expected closed, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 2)
	cb.Apply()
	defer cb.Revert()

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.GetState())
	}
	if cb.Allow() {
		t.Fatal("circuit breaker should block in open state")
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 2)
	cb.Apply()
	defer cb.Revert()

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Fatalf("expected open, got %s", cb.GetState())
	}

	cb.RecordSuccess()
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.GetState())
	}

	cb.RecordSuccess()
	if cb.GetState() != StateClosed {
		t.Fatalf("expected closed after recovery, got %s", cb.GetState())
	}
	if !cb.Allow() {
		t.Fatal("circuit breaker should allow after closing")
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(2, 3)
	cb.Apply()
	defer cb.Revert()

	cb.RecordFailure()
	cb.RecordFailure()

	cb.RecordSuccess()
	if cb.GetState() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.GetState())
	}

	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Fatalf("expected open after half-open failure, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(5, 3)
	cb.Apply()
	defer cb.Revert()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if cb.Allow() {
					if j%3 == 0 {
						cb.RecordFailure()
					} else {
						cb.RecordSuccess()
					}
				}
			}
		}(i)
	}
	wg.Wait()

	state := cb.GetState()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Fatalf("invalid state: %v", state)
	}
}

func TestCircuitBreaker_Duration(t *testing.T) {
	cb := NewCircuitBreaker(3, 2)
	cb.Apply()
	time.Sleep(15 * time.Millisecond)
	cb.Revert()

	d := cb.Duration()
	if d < 10*time.Millisecond {
		t.Fatalf("expected duration >= 10ms, got %v", d)
	}
}

func TestCircuitBreaker_AllowWhenNotApplied(t *testing.T) {
	cb := NewCircuitBreaker(1, 1)
	if !cb.Allow() {
		t.Fatal("Allow should return true when not applied")
	}
}

// ── Scenario Tests ───────────────────────────────

func TestScenario_ApplyRevert(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	re := NewResourceExhaustion("disk-full", 500, 0)
	li := NewLatencyInjection(5 * time.Millisecond)

	s := NewScenario(np, re, li)
	if err := s.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !np.IsApplied() {
		t.Fatal("network partition should be applied")
	}
	if !re.IsApplied() {
		t.Fatal("resource exhaustion should be applied")
	}
	if !li.IsApplied() {
		t.Fatal("latency injection should be applied")
	}

	if err := s.Revert(); err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if np.IsApplied() || re.IsApplied() || li.IsApplied() {
		t.Fatal("all faults should be reverted")
	}
}

func TestScenario_CompositeChaos(t *testing.T) {
	cb := NewCircuitBreaker(2, 1)
	pk := NewProcessKill(42, 1*time.Second)

	s := NewScenario(cb, pk)
	s.Apply()
	defer s.Revert()

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Fatalf("expected open, got %s", cb.GetState())
	}
}

func TestScenario_ActiveFaults(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	li := NewLatencyInjection(5 * time.Millisecond)

	s := NewScenario(np, li)
	if s.ActiveFaults() != 0 {
		t.Fatal("expected 0 active faults before apply")
	}

	s.Apply()
	if s.ActiveFaults() != 2 {
		t.Fatalf("expected 2 active faults, got %d", s.ActiveFaults())
	}

	s.Revert()
	if s.ActiveFaults() != 0 {
		t.Fatal("expected 0 active faults after revert")
	}
}

func TestScenario_Run(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	s := NewScenario(np)

	report, err := s.Run(context.Background(), 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if np.IsApplied() {
		t.Fatal("fault should be reverted after Run")
	}
	if report.RecoveryTimeMs < 0 {
		t.Fatal("recovery time should be non-negative")
	}
}

func TestScenario_DoubleApply(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	s := NewScenario(np)
	s.Apply()

	if err := s.Apply(); err != ErrAlreadyApplied {
		t.Fatalf("expected ErrAlreadyApplied, got %v", err)
	}
	s.Revert()
}

func TestScenario_DoubleRevert(t *testing.T) {
	np := NewNetworkPartition([]string{"10.0.0.1"}, nil)
	s := NewScenario(np)
	s.Apply()
	s.Revert()

	if err := s.Revert(); err != ErrNotApplied {
		t.Fatalf("expected ErrNotApplied, got %v", err)
	}
}

// ── AbortController Tests ────────────────────────

func TestAbortController_Abort(t *testing.T) {
	ac := NewAbortController()
	if ac.IsAborted() {
		t.Error("should not be aborted initially")
	}

	ac.Abort()
	if !ac.IsAborted() {
		t.Error("should be aborted after Abort()")
	}

	select {
	case <-ac.Done():
		// ok
	default:
		t.Error("Done channel should be closed after Abort()")
	}
}

func TestAbortController_DoubleAbort(t *testing.T) {
	ac := NewAbortController()
	ac.Abort()
	ac.Abort() // should not panic
}

// ── Builder helper tests ─────────────────────────

func TestBuilderHelpers(t *testing.T) {
	s1 := NetworkPartitionScenario("test", []string{"10.0.0.1"}, []string{"db"}, 10*time.Second)
	if s1 == nil {
		t.Fatal("NetworkPartitionScenario returned nil")
	}

	s2 := LatencyScenario("test", 10*time.Millisecond, 50*time.Millisecond)
	if s2 == nil {
		t.Fatal("LatencyScenario returned nil")
	}

	s3 := ErrorScenario("test", 0.5, "503")
	if s3 == nil {
		t.Fatal("ErrorScenario returned nil")
	}

	s4 := ResourceExhaustionScenario("test", "oom", 1024*1024, 0)
	if s4 == nil {
		t.Fatal("ResourceExhaustionScenario returned nil")
	}

	s5 := DiskFillScenario("test", "/tmp", 1024*1024)
	if s5 == nil {
		t.Fatal("DiskFillScenario returned nil")
	}

	s6 := CompositeScenario([]string{"10.0.0.1"}, 10*time.Millisecond, 50*time.Millisecond, 0.1, "503")
	if s6 == nil {
		t.Fatal("CompositeScenario returned nil")
	}
}
