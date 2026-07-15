// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — registry.go centralizes adapter lookup and discovery.
package tier

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// Adapter is the public contract every tier in pkg/tier must satisfy.
//
// Implementations are expected to be safe to call from multiple goroutines
// (e.g. concurrent Deploy/Stop). The Deploy, Start, Stop, Delete and
// GetStartupTime methods are required; Info and Probe have useful defaults
// supplied by *BaseAdapter-style helpers but may be overridden.
type Adapter interface {
	// Deploy allocates a new sandbox for the given config and returns its
	// descriptor. The adapter may still need Start to actually launch it.
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)

	// Start launches the sandbox identified by id.
	Start(ctx context.Context, id string) error

	// Stop terminates the sandbox identified by id.
	Stop(ctx context.Context, id string) error

	// Delete releases all resources associated with id.
	Delete(ctx context.Context, id string) error

	// GetStartupTime reports the typical cold-start latency for this tier.
	GetStartupTime() time.Duration

	// Info returns static metadata used by `nvms tier info <name>`.
	Info() TierInfo

	// Probe checks whether the underlying runtime is available on the
	// current host. It MUST NOT actually launch anything; it only inspects
	// the environment (kernel support, $PATH binary, etc.).
	Probe(ctx context.Context) error
}

// TierInfo is the static, descriptive metadata for a tier. It powers both
// `nvms tier list` and `nvms tier info <name>` and is part of the public
// API consumers can rely on for auto-selection.
type TierInfo struct {
	// Name is the canonical lowercase short name (e.g. "firecracker").
	Name string

	// Number is the legacy integer selector kept for backwards
	// compatibility with the original 3-tier flag (--tier 1/2/3). Tiers
	// introduced after the initial trio get number 0.
	Number int

	// DisplayName is a human-friendly label.
	DisplayName string

	// Description is a one-line summary of the tier.
	Description string

	// Startup is the typical cold-start latency, in milliseconds.
	StartupMS int

	// MemoryMB is the per-instance memory overhead, in MiB.
	MemoryMB int

	// Security is the level of isolation the tier provides
	// (low / medium / high / untrusted).
	Security string

	// Platform lists the operating systems this tier supports
	// (linux, macos, windows). Empty means "any".
	Platforms []string

	// Workloads hints at the workload classes this tier is well-suited for
	// (browser, code, tool, cli, etc.). Empty means "any".
	Workloads []string
}

// Registry maps tier names to Adapters. The zero value is NOT usable; use
// NewRegistry.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	// infoByName is redundant with adapters[name].Info() but caches the
	// static info so List() can be O(n) without re-deriving it.
	infoByName map[string]TierInfo
}

// NewRegistry returns an empty Registry. Callers should populate it with
// Register or use DefaultRegistry for the full set of 15+ tiers.
func NewRegistry() *Registry {
	return &Registry{
		adapters:   make(map[string]Adapter),
		infoByName: make(map[string]TierInfo),
	}
}

// Register adds an adapter to the registry under the given name. The name
// must be non-empty; it is stored verbatim (callers are expected to use
// lowercase canonical names). Duplicate names are rejected.
func (r *Registry) Register(name string, adapter Adapter) error {
	if name == "" {
		return fmt.Errorf("tier: empty name")
	}
	if adapter == nil {
		return fmt.Errorf("tier %q: nil adapter", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("tier %q: already registered", name)
	}
	r.adapters[name] = adapter
	r.infoByName[name] = adapter.Info()
	return nil
}

// MustRegister is the panic-on-error variant of Register. Useful for init-time
// wiring in DefaultRegistry.
func (r *Registry) MustRegister(name string, adapter Adapter) {
	if err := r.Register(name, adapter); err != nil {
		panic(err)
	}
}

// Get returns the adapter registered under name, or an error listing all
// available names if not found.
func (r *Registry) Get(name string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("tier %q: not found (registered: %s)", name, joinNames(r.adapters))
	}
	return a, nil
}

// List returns the registered tier names in deterministic (sorted) order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.adapters))
	for k := range r.adapters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Info returns a snapshot of all tier metadata keyed by name. Safe to call
// from any goroutine.
func (r *Registry) Info() map[string]TierInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]TierInfo, len(r.infoByName))
	for k, v := range r.infoByName {
		out[k] = v
	}
	return out
}

// Has reports whether name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.adapters[name]
	return ok
}

// joinNames returns a comma-separated sorted list of registered names for
// use in "not found" error messages.
func joinNames(m map[string]Adapter) string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// AdapterEntry pairs an adapter with the name it was registered under.
// Returned by RegisterAll so callers can iterate without re-querying.
type AdapterEntry struct {
	Name    string
	Adapter Adapter
}

// RegisterAll bulk-registers a slice of {name, adapter} pairs. It is the
// inverse of List and is intended for tests and embedders that want to
// wire a custom registry in one call. Registration stops at the first
// duplicate, empty name, or nil adapter; the error names the offending
// entry.
func (r *Registry) RegisterAll(entries []AdapterEntry) error {
	for i, e := range entries {
		if err := r.Register(e.Name, e.Adapter); err != nil {
			return fmt.Errorf("tier: RegisterAll entry %d (%q): %w", i, e.Name, err)
		}
	}
	return nil
}

// Names is an alias for List, kept for callers that prefer the shorter
// name. Returns the registered tier names in deterministic sorted order.
func (r *Registry) Names() []string { return r.List() }

// BySecurity returns the names of all registered tiers whose Security
// matches the given level (low/medium/high/untrusted). The result is
// sorted alphabetically.
func (r *Registry) BySecurity(security string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.infoByName))
	for n, info := range r.infoByName {
		if strings.EqualFold(info.Security, security) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ByPlatform returns the names of all registered tiers that advertise
// support for the given platform (linux/macos/windows). An empty
// Platforms list is treated as "any" and always matches. Result is
// sorted alphabetically.
func (r *Registry) ByPlatform(platform string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.infoByName))
	for n, info := range r.infoByName {
		if platformMatches(info.Platforms, Platform(platform)) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ByStartup returns the names of all registered tiers whose StartupMS
// is <= maxMS. Used by policy.go to honor a startup budget. Result is
// sorted alphabetically.
func (r *Registry) ByStartup(maxMS int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.infoByName))
	for n, info := range r.infoByName {
		if info.StartupMS <= maxMS {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// defaultRegistryOnce guards the lazy initialization of the default registry.
var defaultRegistryOnce sync.Once

// defaultRegistry holds the package-wide registry returned by DefaultRegistry.
var defaultRegistry *Registry

// DefaultRegistry returns a registry pre-populated with the full set of
// 15 tiers. Construction is lazy and idempotent; the result is safe to
// share across goroutines.
//
// All adapters are constructed best-effort: tiers whose runtime is not
// available on the current host are still registered. They will be visible
// in List/Info, but Probe() will return an error.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		r := NewRegistry()
		registerDefaultTiers(r)
		defaultRegistry = r
	})
	return defaultRegistry
}
