// SPDX-License-Identifier: MIT OR Apache-2.0
// Package chaos provides simulation types for chaos engineering tests.
//
// It models common failure modes (network partitions, resource exhaustion,
// process kills, latency injection, error injection, disk fill, and circuit
// breakers) and provides Apply/Revert methods so tests can inject and clean up
// fault conditions.  A Scenario orchestrator runs multiple faults as a
// composite chaos experiment with impact reporting.
package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Error Sentinels
// ---------------------------------------------------------------------------

var (
	ErrAlreadyApplied = errors.New("chaos: fault already applied")
	ErrNotApplied     = errors.New("chaos: fault not applied")
	ErrReverted       = errors.New("chaos: already reverted")
)

// ---------------------------------------------------------------------------
// Fault interface
// ---------------------------------------------------------------------------

// Fault represents a single injectable chaos condition.
type Fault interface {
	// Apply activates the fault condition.
	Apply() error
	// Revert deactivates the fault and restores normal behavior.
	Revert() error
	// IsApplied returns true if the fault is currently active.
	IsApplied() bool
	// Name returns a human-readable label for the fault.
	Name() string
}

// ---------------------------------------------------------------------------
// NetworkPartition
// ---------------------------------------------------------------------------

// NetworkPartition simulates a network partition by blocking connections
// between specified endpoints.  Uses an allowlist/blocklist model.
type NetworkPartition struct {
	mu           sync.Mutex
	applied      bool
	BlockedIP    []string
	BlockedHosts []string
	applyTime    time.Time
	revertTime   time.Time
}

// NewNetworkPartition creates a NetworkPartition blocking the given IPs/hosts.
func NewNetworkPartition(blockedIPs []string, blockedHosts []string) *NetworkPartition {
	return &NetworkPartition{
		BlockedIP:    blockedIPs,
		BlockedHosts: blockedHosts,
	}
}

func (np *NetworkPartition) Name() string { return "network-partition" }

func (np *NetworkPartition) Apply() error {
	np.mu.Lock()
	defer np.mu.Unlock()
	if np.applied {
		return ErrAlreadyApplied
	}
	np.applied = true
	np.applyTime = time.Now()
	return nil
}

func (np *NetworkPartition) Revert() error {
	np.mu.Lock()
	defer np.mu.Unlock()
	if !np.applied {
		return ErrNotApplied
	}
	np.applied = false
	np.revertTime = time.Now()
	return nil
}

func (np *NetworkPartition) IsApplied() bool {
	np.mu.Lock()
	defer np.mu.Unlock()
	return np.applied
}

// IsBlocked checks whether a given IP or hostname is blocked.
func (np *NetworkPartition) IsBlocked(ipOrHost string) bool {
	np.mu.Lock()
	defer np.mu.Unlock()
	if !np.applied {
		return false
	}
	for _, ip := range np.BlockedIP {
		if ip == ipOrHost {
			return true
		}
	}
	for _, host := range np.BlockedHosts {
		if host == ipOrHost {
			return true
		}
	}
	return false
}

// Duration returns how long the partition was active.
func (np *NetworkPartition) Duration() time.Duration {
	np.mu.Lock()
	defer np.mu.Unlock()
	if !np.applied && np.revertTime.IsZero() {
		return 0
	}
	end := np.revertTime
	if np.applied {
		end = time.Now()
	}
	return end.Sub(np.applyTime)
}

// ---------------------------------------------------------------------------
// ResourceExhaustion
// ---------------------------------------------------------------------------

// ResourceExhaustion simulates OOM, disk-full, or fd-exhaustion conditions.
type ResourceExhaustion struct {
	mu          sync.Mutex
	applied     bool
	Type        string // "oom", "disk-full", "fd-exhaustion"
	BudgetBytes int64
	BudgetFDs   int
	spentBytes  int64
	openFDs     int
	applyTime   time.Time
	revertTime  time.Time
}

// NewResourceExhaustion creates a ResourceExhaustion of the given type.
func NewResourceExhaustion(resourceType string, budgetBytes int64, budgetFDs int) *ResourceExhaustion {
	return &ResourceExhaustion{
		Type:        resourceType,
		BudgetBytes: budgetBytes,
		BudgetFDs:   budgetFDs,
	}
}

func (re *ResourceExhaustion) Name() string {
	return fmt.Sprintf("resource-exhaustion(%s)", re.Type)
}

func (re *ResourceExhaustion) Apply() error {
	re.mu.Lock()
	defer re.mu.Unlock()
	if re.applied {
		return ErrAlreadyApplied
	}
	re.applied = true
	re.applyTime = time.Now()
	return nil
}

func (re *ResourceExhaustion) Revert() error {
	re.mu.Lock()
	defer re.mu.Unlock()
	if !re.applied {
		return ErrNotApplied
	}
	re.applied = false
	re.revertTime = time.Now()
	re.spentBytes = 0
	re.openFDs = 0
	return nil
}

func (re *ResourceExhaustion) IsApplied() bool {
	re.mu.Lock()
	defer re.mu.Unlock()
	return re.applied
}

// ConsumeBytes tries to consume bytes from the budget.
func (re *ResourceExhaustion) ConsumeBytes(n int64) error {
	re.mu.Lock()
	defer re.mu.Unlock()
	if !re.applied {
		return nil
	}
	if re.spentBytes+n > re.BudgetBytes {
		return fmt.Errorf("%s: budget exhausted (%d + %d > %d bytes)",
			re.Type, re.spentBytes, n, re.BudgetBytes)
	}
	re.spentBytes += n
	return nil
}

// AllocateFD tries to allocate a file descriptor from the budget.
func (re *ResourceExhaustion) AllocateFD() error {
	re.mu.Lock()
	defer re.mu.Unlock()
	if !re.applied {
		return nil
	}
	if re.openFDs >= re.BudgetFDs {
		return fmt.Errorf("%s: fd limit reached (%d >= %d)",
			re.Type, re.openFDs, re.BudgetFDs)
	}
	re.openFDs++
	return nil
}

// FreeFD releases a file descriptor.
func (re *ResourceExhaustion) FreeFD() {
	re.mu.Lock()
	defer re.mu.Unlock()
	if re.openFDs > 0 {
		re.openFDs--
	}
}

// RemainingBudget returns the remaining byte budget.
func (re *ResourceExhaustion) RemainingBudget() int64 {
	re.mu.Lock()
	defer re.mu.Unlock()
	return re.BudgetBytes - re.spentBytes
}

// ---------------------------------------------------------------------------
// ProcessKill
// ---------------------------------------------------------------------------

// ProcessKill simulates killing a process at a specified time or event.
type ProcessKill struct {
	mu        sync.Mutex
	applied   bool
	PID       int
	Delay     time.Duration
	killed    bool
	applyTime time.Time
	cancel    context.CancelFunc
}

// NewProcessKill creates a ProcessKill for the given PID with a delay.
func NewProcessKill(pid int, delay time.Duration) *ProcessKill {
	return &ProcessKill{
		PID:   pid,
		Delay: delay,
	}
}

func (pk *ProcessKill) Name() string {
	return fmt.Sprintf("process-kill(pid=%d)", pk.PID)
}

func (pk *ProcessKill) Apply() error {
	pk.mu.Lock()
	defer pk.mu.Unlock()
	if pk.applied {
		return ErrAlreadyApplied
	}
	pk.applied = true
	pk.applyTime = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	pk.cancel = cancel

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pk.Delay):
			pk.mu.Lock()
			pk.killed = true
			pk.mu.Unlock()
		}
	}()

	return nil
}

func (pk *ProcessKill) Revert() error {
	pk.mu.Lock()
	defer pk.mu.Unlock()
	if !pk.applied {
		return ErrNotApplied
	}
	if pk.cancel != nil {
		pk.cancel()
	}
	pk.applied = false
	pk.killed = false
	return nil
}

func (pk *ProcessKill) IsApplied() bool {
	pk.mu.Lock()
	defer pk.mu.Unlock()
	return pk.applied
}

// IsKilled returns true if the kill delay has elapsed.
func (pk *ProcessKill) IsKilled() bool {
	pk.mu.Lock()
	defer pk.mu.Unlock()
	return pk.killed
}

// Wait blocks until the process kill has fired or is cancelled.
func (pk *ProcessKill) Wait(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if pk.IsKilled() {
				return nil
			}
		}
	}
}

// ---------------------------------------------------------------------------
// LatencyInjection
// ---------------------------------------------------------------------------

// LatencyInjection adds artificial delay to operations.
type LatencyInjection struct {
	mu         sync.Mutex
	applied    bool
	MinDelay   time.Duration
	MaxDelay   time.Duration
	applyTime  time.Time
	revertTime time.Time
}

// NewLatencyInjection creates a LatencyInjection with a fixed delay.
func NewLatencyInjection(delay time.Duration) *LatencyInjection {
	return &LatencyInjection{
		MinDelay: delay,
		MaxDelay: delay,
	}
}

// NewRangeLatencyInjection creates a LatencyInjection with random delay
// in the range [min, max].
func NewRangeLatencyInjection(minDelay, maxDelay time.Duration) *LatencyInjection {
	return &LatencyInjection{
		MinDelay: minDelay,
		MaxDelay: maxDelay,
	}
}

func (li *LatencyInjection) Name() string { return "latency-injection" }

func (li *LatencyInjection) Apply() error {
	li.mu.Lock()
	defer li.mu.Unlock()
	if li.applied {
		return ErrAlreadyApplied
	}
	li.applied = true
	li.applyTime = time.Now()
	return nil
}

func (li *LatencyInjection) Revert() error {
	li.mu.Lock()
	defer li.mu.Unlock()
	if !li.applied {
		return ErrNotApplied
	}
	li.applied = false
	li.revertTime = time.Now()
	return nil
}

func (li *LatencyInjection) IsApplied() bool {
	li.mu.Lock()
	defer li.mu.Unlock()
	return li.applied
}

// Wait blocks for a duration between MinDelay and MaxDelay.
// No-op when the fault is not applied.
func (li *LatencyInjection) Wait() {
	li.mu.Lock()
	applied := li.applied
	minD := li.MinDelay
	maxD := li.MaxDelay
	li.mu.Unlock()

	if !applied {
		return
	}
	if minD == maxD {
		time.Sleep(minD)
		return
	}
	delta := maxD - minD
	delay := minD + time.Duration(rand.Int63n(int64(delta)))
	time.Sleep(delay)
}

// Duration returns how long the latency injection has been active.
func (li *LatencyInjection) Duration() time.Duration {
	li.mu.Lock()
	defer li.mu.Unlock()
	if !li.applied && li.revertTime.IsZero() {
		return 0
	}
	end := li.revertTime
	if li.applied {
		end = time.Now()
	}
	return end.Sub(li.applyTime)
}

// ---------------------------------------------------------------------------
// ErrorInjection
// ---------------------------------------------------------------------------

// ErrorInjection simulates random failures at a configurable rate.
type ErrorInjection struct {
	mu        sync.Mutex
	applied   bool
	ErrorRate float64 // probability [0,1] that a request fails
	ErrorType string  // "timeout", "connection_refused", "503", "custom"
	rng       *rand.Rand
	fails     int64
	successes int64
	applyTime time.Time
	revertTime time.Time
}

// NewErrorInjection creates an ErrorInjection with the given rate and type.
func NewErrorInjection(rate float64, errorType string) *ErrorInjection {
	return &ErrorInjection{
		ErrorRate: rate,
		ErrorType: errorType,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (ei *ErrorInjection) Name() string {
	return fmt.Sprintf("error-injection(%s,%.0f%%)", ei.ErrorType, ei.ErrorRate*100)
}

func (ei *ErrorInjection) Apply() error {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	if ei.applied {
		return ErrAlreadyApplied
	}
	ei.applied = true
	ei.applyTime = time.Now()
	return nil
}

func (ei *ErrorInjection) Revert() error {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	if !ei.applied {
		return ErrNotApplied
	}
	ei.applied = false
	ei.revertTime = time.Now()
	return nil
}

func (ei *ErrorInjection) IsApplied() bool {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	return ei.applied
}

// ShouldFail returns true if this request should be failed based on the
// configured error rate.  Returns false when not applied.
func (ei *ErrorInjection) ShouldFail() bool {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	if !ei.applied {
		return false
	}
	fail := ei.rng.Float64() < ei.ErrorRate
	if fail {
		ei.fails++
	} else {
		ei.successes++
	}
	return fail
}

// Error returns the appropriate error type for this injection.
func (ei *ErrorInjection) Error() error {
	switch ei.ErrorType {
	case "timeout":
		return context.DeadlineExceeded
	case "connection_refused":
		return errors.New("connection refused")
	case "connection_reset":
		return errors.New("connection reset by peer")
	case "503":
		return errors.New("service unavailable (503)")
	case "500":
		return errors.New("internal server error (500)")
	default:
		return fmt.Errorf("chaos: injected %s error", ei.ErrorType)
	}
}

// Stats returns the number of failed and successful requests.
func (ei *ErrorInjection) Stats() (fails, successes int64) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	return ei.fails, ei.successes
}

// ---------------------------------------------------------------------------
// DiskFill
// ---------------------------------------------------------------------------

// DiskFill simulates filling a disk path with data up to a target size.
// This is a logical simulation – it tracks how much would be written.
type DiskFill struct {
	mu          sync.Mutex
	applied     bool
	Path        string // logical path being filled
	TargetBytes int64  // target fill amount
	filledBytes int64
	applyTime   time.Time
	revertTime  time.Time
}

// NewDiskFill creates a DiskFill for the given path and target size.
func NewDiskFill(path string, targetBytes int64) *DiskFill {
	return &DiskFill{
		Path:        path,
		TargetBytes: targetBytes,
	}
}

func (df *DiskFill) Name() string {
	return fmt.Sprintf("disk-fill(%s,%dbytes)", df.Path, df.TargetBytes)
}

func (df *DiskFill) Apply() error {
	df.mu.Lock()
	defer df.mu.Unlock()
	if df.applied {
		return ErrAlreadyApplied
	}
	df.applied = true
	df.applyTime = time.Now()
	return nil
}

func (df *DiskFill) Revert() error {
	df.mu.Lock()
	defer df.mu.Unlock()
	if !df.applied {
		return ErrNotApplied
	}
	df.applied = false
	df.revertTime = time.Now()
	df.filledBytes = 0
	return nil
}

func (df *DiskFill) IsApplied() bool {
	df.mu.Lock()
	defer df.mu.Unlock()
	return df.applied
}

// Write simulates writing n bytes to the filled path.
// Returns an error when the target is reached.
func (df *DiskFill) Write(n int64) error {
	df.mu.Lock()
	defer df.mu.Unlock()
	if !df.applied {
		return nil
	}
	if df.filledBytes+n > df.TargetBytes {
		return fmt.Errorf("disk fill: no space left on device (%s)", df.Path)
	}
	df.filledBytes += n
	return nil
}

// FilledBytes returns the amount of data written so far.
func (df *DiskFill) FilledBytes() int64 {
	df.mu.Lock()
	defer df.mu.Unlock()
	return df.filledBytes
}

// ---------------------------------------------------------------------------
// CircuitBreaker
// ---------------------------------------------------------------------------

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	StateClosed   CircuitBreakerState = iota // Normal operation
	StateOpen                               // Failing all requests
	StateHalfOpen                           // Testing if upstream recovered
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker simulates a circuit breaker pattern.
type CircuitBreaker struct {
	mu               sync.Mutex
	applied          bool
	state            CircuitBreakerState
	failureThreshold int
	successThreshold int
	consecutiveFails int
	consecutiveOKs   int
	halfOpenMax      int
	halfOpenCount    int
	applyTime        time.Time
	revertTime       time.Time
}

// NewCircuitBreaker creates a CircuitBreaker with the given thresholds.
func NewCircuitBreaker(failureThreshold, successThreshold int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		halfOpenMax:      1,
	}
}

func (cb *CircuitBreaker) Name() string { return "circuit-breaker" }

func (cb *CircuitBreaker) Apply() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.applied {
		return ErrAlreadyApplied
	}
	cb.applied = true
	cb.applyTime = time.Now()
	return nil
}

func (cb *CircuitBreaker) Revert() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.applied {
		return ErrNotApplied
	}
	cb.applied = false
	cb.revertTime = time.Now()
	cb.state = StateClosed
	cb.consecutiveFails = 0
	cb.consecutiveOKs = 0
	cb.halfOpenCount = 0
	return nil
}

func (cb *CircuitBreaker) IsApplied() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.applied
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// RecordSuccess records a successful operation and may transition state.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.applied {
		return
	}

	switch cb.state {
	case StateOpen:
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
		cb.consecutiveOKs = 1
	case StateHalfOpen:
		cb.consecutiveOKs++
		cb.halfOpenCount++
		if cb.consecutiveOKs >= cb.successThreshold || cb.halfOpenCount >= cb.halfOpenMax {
			cb.state = StateClosed
			cb.consecutiveFails = 0
			cb.consecutiveOKs = 0
			cb.halfOpenCount = 0
		}
	case StateClosed:
		cb.consecutiveFails = 0
		cb.consecutiveOKs = 0
	}
}

// RecordFailure records a failed operation and may transition state.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.applied {
		return
	}

	switch cb.state {
	case StateClosed:
		cb.consecutiveFails++
		if cb.consecutiveFails >= cb.failureThreshold {
			cb.state = StateOpen
			cb.consecutiveFails = 0
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.consecutiveOKs = 0
		cb.halfOpenCount = 0
	case StateOpen:
		// Already open
	}
}

// Allow returns true if the circuit breaker allows the operation to proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.applied {
		return true
	}
	return cb.state != StateOpen
}

// Duration returns how long the circuit breaker has been active.
func (cb *CircuitBreaker) Duration() time.Duration {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.applied && cb.revertTime.IsZero() {
		return 0
	}
	end := cb.revertTime
	if cb.applied {
		end = time.Now()
	}
	return end.Sub(cb.applyTime)
}

// ---------------------------------------------------------------------------
// Impact report
// ---------------------------------------------------------------------------

// ImpactReport quantifies the blast radius of a chaos experiment.
type ImpactReport struct {
	RequestsFailed   int64   `json:"requests_failed"`
	RequestsAffected int64   `json:"requests_affected"`
	LatencyIncrease  float64 `json:"latency_increase_ms"`
	ErrorRate        float64 `json:"error_rate"`
	RecoveryTimeMs   int64   `json:"recovery_time_ms"`
}

// ---------------------------------------------------------------------------
// Scenario orchestrator
// ---------------------------------------------------------------------------

// Scenario bundles multiple faults together for composite chaos testing.
type Scenario struct {
	mu      sync.Mutex
	Faults  []Fault
	applied bool
}

// NewScenario creates a Scenario with the given faults.
func NewScenario(faults ...Fault) *Scenario {
	return &Scenario{Faults: faults}
}

func (s *Scenario) Name() string { return "composite-scenario" }

func (s *Scenario) Apply() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.applied {
		return ErrAlreadyApplied
	}
	for _, f := range s.Faults {
		if err := f.Apply(); err != nil {
			return fmt.Errorf("scenario apply failed on %s: %w", f.Name(), err)
		}
	}
	s.applied = true
	return nil
}

func (s *Scenario) Revert() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.applied {
		return ErrNotApplied
	}
	var firstErr error
	for _, f := range s.Faults {
		if err := f.Revert(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("scenario revert failed on %s: %w", f.Name(), err)
		}
	}
	s.applied = false
	return firstErr
}

func (s *Scenario) IsApplied() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applied
}

// ActiveFaults returns the count of currently applied faults.
func (s *Scenario) ActiveFaults() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.applied {
		return 0
	}
	count := 0
	for _, f := range s.Faults {
		if f.IsApplied() {
			count++
		}
	}
	return count
}

// Run runs the scenario for the given duration, then reverts.
// Returns the impact report.
func (s *Scenario) Run(ctx context.Context, duration time.Duration) (ImpactReport, error) {
	if err := s.Apply(); err != nil {
		return ImpactReport{}, err
	}

	select {
	case <-ctx.Done():
	case <-time.After(duration):
	}

	start := time.Now()
	err := s.Revert()
	recoveryMs := time.Since(start).Milliseconds()

	return ImpactReport{
		RecoveryTimeMs: recoveryMs,
	}, err
}

// ---------------------------------------------------------------------------
// Scenario builder helpers
// ---------------------------------------------------------------------------

// NetworkPartitionScenario creates a standard network partition scenario.
func NetworkPartitionScenario(name string, blockedIPs []string, blockedHosts []string, duration time.Duration) *Scenario {
	return NewScenario(NewNetworkPartition(blockedIPs, blockedHosts))
}

// LatencyScenario creates a latency injection scenario.
func LatencyScenario(name string, minDelay, maxDelay time.Duration) *Scenario {
	return NewScenario(NewRangeLatencyInjection(minDelay, maxDelay))
}

// ErrorScenario creates an error injection scenario.
func ErrorScenario(name string, rate float64, errorType string) *Scenario {
	return NewScenario(NewErrorInjection(rate, errorType))
}

// ResourceExhaustionScenario creates a resource exhaustion scenario.
func ResourceExhaustionScenario(name string, resourceType string, budgetBytes int64, budgetFDs int) *Scenario {
	return NewScenario(NewResourceExhaustion(resourceType, budgetBytes, budgetFDs))
}

// DiskFillScenario creates a disk fill scenario.
func DiskFillScenario(name string, path string, targetBytes int64) *Scenario {
	return NewScenario(NewDiskFill(path, targetBytes))
}

// CompositeScenario creates a scenario with network partition + latency + errors.
func CompositeScenario(
	blockedIPs []string,
	latencyMin, latencyMax time.Duration,
	errorRate float64,
	errorType string,
) *Scenario {
	return NewScenario(
		NewNetworkPartition(blockedIPs, nil),
		NewRangeLatencyInjection(latencyMin, latencyMax),
		NewErrorInjection(errorRate, errorType),
	)
}

// ---------------------------------------------------------------------------
// Abort controller
// ---------------------------------------------------------------------------

// AbortController allows external cancellation of chaos injection.
type AbortController struct {
	mu      sync.Mutex
	aborted bool
	ch      chan struct{}
}

// NewAbortController creates a new AbortController.
func NewAbortController() *AbortController {
	return &AbortController{
		ch: make(chan struct{}),
	}
}

// Abort signals all injected chaos to stop immediately.
func (ac *AbortController) Abort() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if !ac.aborted {
		ac.aborted = true
		close(ac.ch)
	}
}

// Done returns a channel that is closed when Abort is called.
func (ac *AbortController) Done() <-chan struct{} {
	return ac.ch
}

// IsAborted returns true if abort has been signalled.
func (ac *AbortController) IsAborted() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.aborted
}
