// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox provides the adapter registry for selecting sandbox backends.
package sandbox

import (
	"fmt"
	"sync"
)

// AdapterType identifies the sandbox adapter backend.
type AdapterType string

const (
	AdapterDocker      AdapterType = "docker"
	AdapterPodman      AdapterType = "podman"
	AdapterGVisor      AdapterType = "gvisor"
	AdapterFirecracker AdapterType = "firecracker"
	AdapterKata        AdapterType = "kata"
	AdapterContainers  AdapterType = "containers"
	AdapterOSV         AdapterType = "osv"
	AdapterUnikernel   AdapterType = "unikernel"
	AdapterLinux       AdapterType = "linux"
	AdapterMac         AdapterType = "mac"
	AdapterProcess     AdapterType = "process"
)

// AdapterCapabilities describes what an adapter can do.
type AdapterCapabilities struct {
	Type           AdapterType
	SupportsGPU    bool
	SupportsFS     bool
	MaxMemoryMB    int
	MaxCPU         int
	RequiresRoot   bool
	IsolationLevel string // "process", "container", "vm"
}

// AdapterRegistry manages sandbox adapter discovery and selection.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[AdapterType]AdapterCapabilities
}

// NewAdapterRegistry creates a registry with built-in adapters.
func NewAdapterRegistry() *AdapterRegistry {
	r := &AdapterRegistry{
		adapters: make(map[AdapterType]AdapterCapabilities),
	}
	r.registerBuiltins()
	return r
}

func (r *AdapterRegistry) registerBuiltins() {
	builtins := []AdapterCapabilities{
		{Type: AdapterDocker, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 8192, MaxCPU: 16, RequiresRoot: false, IsolationLevel: "container"},
		{Type: AdapterPodman, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 8192, MaxCPU: 16, RequiresRoot: false, IsolationLevel: "container"},
		{Type: AdapterGVisor, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 4096, MaxCPU: 8, RequiresRoot: false, IsolationLevel: "container"},
		{Type: AdapterFirecracker, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 16384, MaxCPU: 32, RequiresRoot: true, IsolationLevel: "vm"},
		{Type: AdapterKata, SupportsGPU: true, SupportsFS: true, MaxMemoryMB: 32768, MaxCPU: 64, RequiresRoot: true, IsolationLevel: "vm"},
		{Type: AdapterContainers, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 4096, MaxCPU: 8, RequiresRoot: false, IsolationLevel: "container"},
		{Type: AdapterOSV, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 2048, MaxCPU: 4, RequiresRoot: false, IsolationLevel: "vm"},
		{Type: AdapterUnikernel, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 1024, MaxCPU: 2, RequiresRoot: false, IsolationLevel: "vm"},
		{Type: AdapterLinux, SupportsGPU: true, SupportsFS: true, MaxMemoryMB: 65536, MaxCPU: 128, RequiresRoot: true, IsolationLevel: "process"},
		{Type: AdapterMac, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 16384, MaxCPU: 16, RequiresRoot: false, IsolationLevel: "process"},
		{Type: AdapterProcess, SupportsGPU: false, SupportsFS: true, MaxMemoryMB: 8192, MaxCPU: 8, RequiresRoot: false, IsolationLevel: "process"},
	}
	for _, b := range builtins {
		r.adapters[b.Type] = b
	}
}

// Register adds or overwrites an adapter.
func (r *AdapterRegistry) Register(cap AdapterCapabilities) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[cap.Type] = cap
}

// Get returns capabilities for the given adapter type.
func (r *AdapterRegistry) Get(adapterType AdapterType) (AdapterCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cap, ok := r.adapters[adapterType]
	if !ok {
		return AdapterCapabilities{}, fmt.Errorf("adapter %q not registered", adapterType)
	}
	return cap, nil
}

// List returns all registered adapter types.
func (r *AdapterRegistry) List() []AdapterType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]AdapterType, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}

// SelectBest returns the most suitable adapter for the given requirements.
func (r *AdapterRegistry) SelectBest(needsGPU bool, memoryMB int, cpus int) (AdapterCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best AdapterCapabilities
	found := false

	for _, cap := range r.adapters {
		if needsGPU && !cap.SupportsGPU {
			continue
		}
		if memoryMB > cap.MaxMemoryMB {
			continue
		}
		if cpus > cap.MaxCPU {
			continue
		}
		if !found || (cap.IsolationLevel == "vm" && best.IsolationLevel != "vm") {
			best = cap
			found = true
		}
	}

	if !found {
		return AdapterCapabilities{}, fmt.Errorf("no adapter found for requirements: gpu=%v mem=%dMB cpus=%d", needsGPU, memoryMB, cpus)
	}
	return best, nil
}
