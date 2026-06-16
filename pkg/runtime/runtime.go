// SPDX-License-Identifier: MIT OR Apache-2.0
// Package runtime provides the runtime interface with Tier1/2/3 implementations.
package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// Runtime is the interface for all tier runtimes.
type Runtime interface {
	// Name returns the runtime name.
	Name() string

	// Tier returns the tier level (1, 2, or 3).
	Tier() int

	// Deploy creates and starts a sandbox workload.
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)

	// StartupTime returns the typical cold-start latency.
	StartupTime() time.Duration
}

// Tier1Runtime implements Runtime for WASM-based Tier1 workloads.
// Startup: ~1ms, Memory: ~1MB, CPU overhead: 0%
type Tier1Runtime struct {
	adapter *tier.WASMAdapter
}

// NewTier1Runtime creates a new Tier1 runtime.
func NewTier1Runtime() *Tier1Runtime {
	return &Tier1Runtime{
		adapter: tier.NewWASMAdapter(),
	}
}

// Name returns the runtime name.
func (r *Tier1Runtime) Name() string {
	return "wasm"
}

// Tier returns the tier level.
func (r *Tier1Runtime) Tier() int {
	return 1
}

// Deploy deploys a Tier1 WASM workload.
func (r *Tier1Runtime) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	return r.adapter.Deploy(ctx, config)
}

// StartupTime returns the typical cold-start latency for Tier1.
func (r *Tier1Runtime) StartupTime() time.Duration {
	return 1 * time.Millisecond
}

// Tier2Runtime implements Runtime for gVisor-based Tier2 workloads.
// Startup: ~90ms, Memory: ~50MB, CPU overhead: ~5%
type Tier2Runtime struct {
	adapter *tier.GVisorAdapter
}

// NewTier2Runtime creates a new Tier2 runtime.
func NewTier2Runtime() *Tier2Runtime {
	return &Tier2Runtime{
		adapter: tier.NewGVisorAdapter(),
	}
}

// Name returns the runtime name.
func (r *Tier2Runtime) Name() string {
	return "gvisor"
}

// Tier returns the tier level.
func (r *Tier2Runtime) Tier() int {
	return 2
}

// Deploy deploys a Tier2 gVisor workload.
func (r *Tier2Runtime) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	return r.adapter.Deploy(ctx, config)
}

// StartupTime returns the typical cold-start latency for Tier2.
func (r *Tier2Runtime) StartupTime() time.Duration {
	return 90 * time.Millisecond
}

// Tier3Runtime implements Runtime for Firecracker-based Tier3 workloads.
// Startup: ~125ms, Memory: ~128MB, CPU overhead: ~10%
type Tier3Runtime struct {
	adapter *tier.FirecrackerAdapter
}

// NewTier3Runtime creates a new Tier3 runtime.
func NewTier3Runtime() *Tier3Runtime {
	return &Tier3Runtime{
		adapter: tier.NewFirecrackerAdapter(),
	}
}

// Name returns the runtime name.
func (r *Tier3Runtime) Name() string {
	return "firecracker"
}

// Tier returns the tier level.
func (r *Tier3Runtime) Tier() int {
	return 3
}

// Deploy deploys a Tier3 Firecracker workload.
func (r *Tier3Runtime) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	return r.adapter.Deploy(ctx, config)
}

// StartupTime returns the typical cold-start latency for Tier3.
func (r *Tier3Runtime) StartupTime() time.Duration {
	return 125 * time.Millisecond
}

// Registry holds all available runtimes.
type Registry struct {
	runtimes map[int]Runtime
}

// NewRegistry creates a new runtime registry with all tiers registered.
func NewRegistry() *Registry {
	return &Registry{
		runtimes: map[int]Runtime{
			1: NewTier1Runtime(),
			2: NewTier2Runtime(),
			3: NewTier3Runtime(),
		},
	}
}

// Get returns the runtime for the specified tier.
func (r *Registry) Get(tier int) (Runtime, error) {
	rt, ok := r.runtimes[tier]
	if !ok {
		return nil, fmt.Errorf("runtime not found for tier %d", tier)
	}
	return rt, nil
}

// Register adds a runtime to the registry.
func (r *Registry) Register(tier int, rt Runtime) {
	if r.runtimes == nil {
		r.runtimes = make(map[int]Runtime)
	}
	r.runtimes[tier] = rt
}

// All returns all registered runtimes.
func (r *Registry) All() []Runtime {
	result := make([]Runtime, 0, len(r.runtimes))
	for _, rt := range r.runtimes {
		result = append(result, rt)
	}
	return result
}

// Ensure interface compliance.
var _ Runtime = (*Tier1Runtime)(nil)
var _ Runtime = (*Tier2Runtime)(nil)
var _ Runtime = (*Tier3Runtime)(nil)
