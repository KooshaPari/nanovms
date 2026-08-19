// Package plugin defines the runtime plugin interface for nanovms.
//
// Plugins extend the daemon with additional backends, adapters, and event
// handlers. They can be:
//   - Linked at build time (in-process)
//   - Loaded at runtime via the standard `plugin` package (Go .so files)
//
// All plugins must satisfy the Plugin interface and provide a registry entry.
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// ID is a stable reverse-DNS plugin identifier.
type ID string

// Info describes a registered plugin (returned at registration time).
type Info struct {
	ID      ID
	Name    string
	Version string
}

// Plugin is the contract every nanovms plugin must satisfy.
//
// Implementations should be safe for concurrent use; the registry
// may invoke Init/Shutdown/Health from multiple goroutines.
type Plugin interface {
	// Info returns static metadata for the plugin.
	Info() Info

	// Init is called once after registration. Use it to set up
	// resources, start background goroutines, etc.
	Init(ctx context.Context) error

	// Shutdown is called once before unload. Release resources here.
	Shutdown(ctx context.Context) error

	// Health returns nil if the plugin is healthy, an error otherwise.
	// Called periodically by the daemon health endpoint.
	Health(ctx context.Context) error
}

// Registry holds loaded plugins and dispatches lifecycle events.
type Registry struct {
	mu      sync.RWMutex
	plugins []Plugin
}

func NewRegistry() *Registry { return &Registry{} }

// Register adds a plugin and calls Init.
func (r *Registry) Register(ctx context.Context, p Plugin) error {
	if p == nil {
		return fmt.Errorf("plugin: nil")
	}
	if err := p.Init(ctx); err != nil {
		return fmt.Errorf("plugin %s init: %w", p.Info().ID, err)
	}
	r.mu.Lock()
	r.plugins = append(r.plugins, p)
	r.mu.Unlock()
	return nil
}

// UnregisterAll calls Shutdown on every registered plugin (LIFO).
func (r *Registry) UnregisterAll(ctx context.Context) {
	r.mu.Lock()
	plugins := r.plugins
	r.plugins = nil
	r.mu.Unlock()
	for i := len(plugins) - 1; i >= 0; i-- {
		_ = plugins[i].Shutdown(ctx)
	}
}

// Find returns the first plugin matching the given ID.
func (r *Registry) Find(id ID) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.plugins {
		if p.Info().ID == id {
			return p, true
		}
	}
	return nil, false
}

// List returns info for all registered plugins.
func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p.Info())
	}
	return out
}

// Len returns the number of registered plugins.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}
