// Package domain contains the core domain models.
package domain

import (
	"fmt"
	"sync/atomic"
)

// WASMRuntime represents the WebAssembly runtime type.
type WASMRuntime string

const (
	WASMRuntimeWasmtime WASMRuntime = "wasmtime"
	WASMRuntimeWasmer   WASMRuntime = "wasmer"
	WASMRuntimeWAVM     WASMRuntime = "wavm"
)

// WASMState represents the state of a WASM instance.
type WASMState string

const (
	WASMStateInstantiated WASMState = "instantiated"
	WASMStateRunning      WASMState = "running"
	WASMStateTerminated   WASMState = "terminated"
)

// CompileOpts holds WASM compilation options.
type CompileOpts struct {
	OptimizeLevel                int  `json:"optimize_level"`
	SupportsStreamingCompilation bool  `json:"supports_streaming"`
	MaxMemoryPages               int  `json:"max_memory_pages"`
}

// ModuleOpts holds WASM module instantiation options.
type ModuleOpts struct {
	MemoryPages int  `json:"memory_pages"`
	ImportFuncs int  `json:"import_funcs"`
	ExportFuncs int  `json:"export_funcs"`
	DebugMode   bool `json:"debug_mode"`
}

// WASMInstance represents a running WASM module instance.
type WASMInstance struct {
	ID      string    `json:"id"`
	Memory  uint64    `json:"memory"`
	State   WASMState `json:"state"`
	Exports []string  `json:"exports,omitempty"`
}

// GenerateID generates a unique ID (simplified - uses atomic counter).
func GenerateID() string {
	id := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("%d", id)
}

var idCounter int64

// SandboxRuntime represents a sandbox runtime implementation.
type SandboxRuntime struct {
	Name    string      `json:"name"`
	Type    SandboxType `json:"type"`
	SubType string      `json:"sub_type,omitempty"`
	Path    string      `json:"path"`
	Version string      `json:"version"`
}

// Additional SandboxType constants for process-level sandboxes.
const (
	SandboxTypeGVisor   SandboxType = "gvisor"
	SandboxTypeLandlock SandboxType = "landlock"
	SandboxTypeWasmtime SandboxType = "wasmtime"
	SandboxTypeSeccomp SandboxType = "seccomp"
)

// Status constants for Sandbox.
const (
	StatusUnknown  SandboxStatus = "unknown"
	StatusCreated  SandboxStatus = "created"
	StatusStarting SandboxStatus = "starting"
	StatusRunning  SandboxStatus = "running"
	StatusStopping SandboxStatus = "stopping"
	StatusStopped  SandboxStatus = "stopped"
	StatusFailed   SandboxStatus = "failed"
)

// SandboxStatusCreating is an alias for StatusStarting (for adapter compatibility).
const SandboxStatusCreating = StatusStarting

// ParseStatus parses a status string into a SandboxStatus.
func ParseStatus(s string) SandboxStatus {
	switch s {
	case "created":
		return StatusCreated
	case "starting":
		return StatusStarting
	case "running":
		return StatusRunning
	case "stopping":
		return StatusStopping
	case "stopped":
		return StatusStopped
	case "failed":
		return StatusFailed
	default:
		return StatusUnknown
	}
}

// VMTier represents the VM tier level (alias for compatibility).
type VMTier = VMFlavor
