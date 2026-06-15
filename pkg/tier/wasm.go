// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier provides public tier adapters for NVMS isolation levels.
package tier

import (
	"context"
	"fmt"

	"github.com/kooshapari/nanovms/internal/adapters/wasm"
	"github.com/kooshapari/nanovms/internal/domain"
)

// WASMAdapter is the Tier1 WASM adapter for lightweight, trusted workloads.
// Startup: ~1ms, Memory: ~1MB, CPU overhead: 0%
type WASMAdapter struct {
	runtime domain.WASMRuntime
	adapter *wasm.WASMAdapter
}

// NewWASMAdapter creates a new Tier1 WASM adapter using wasmtime.
func NewWASMAdapter() *WASMAdapter {
	rt := domain.WASMRuntimeWasmtime
	return &WASMAdapter{
		runtime: rt,
		adapter: wasm.NewWASMAdapter(rt),
	}
}

// Deploy deploys a WASM workload.
func (a *WASMAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.adapter.Probe(ctx); err != nil {
		return nil, fmt.Errorf("WASM runtime not available: %w", err)
	}

	sandbox := &domain.Sandbox{
		ID:     fmt.Sprintf("wasm-%s", domain.GenerateID()),
		Name:   config.Name,
		Status: domain.SandboxStatusRunning,
		Type:   domain.SandboxTypeWasm,
		Config: &config,
	}
	return sandbox, nil
}
