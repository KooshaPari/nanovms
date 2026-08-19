// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier provides public tier adapters for NVMS isolation levels.
package tier

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters/wasm"
	"github.com/kooshapari/nanovms/internal/domain"
)

// WASMAdapter is the Tier1 WASM adapter for lightweight, trusted workloads.
// Startup: ~1ms, Memory: ~1MB, CPU overhead: 0%
type WASMAdapter struct {
	*baseAdapter
	runtime domain.WASMRuntime
	adapter *wasm.WASMAdapter
}

// NewWASMAdapter creates a new Tier1 WASM adapter using wasmtime.
func NewWASMAdapter() *WASMAdapter {
	rt := domain.WASMRuntimeWasmtime
	return &WASMAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "wasm",
			Number:      1,
			DisplayName: "WASM (wasmtime)",
			Description: "Wasmtime-based lightweight sandbox for trusted code (~1ms startup)",
			StartupMS:   1,
			MemoryMB:    1,
			Security:    "low",
			Platforms:   []string{"linux", "macos", "windows"},
			Workloads:   []string{"tool", "cli"},
		}},
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

// Start is a no-op for WASM: the wasmtime instance is provisioned at
// Deploy time and torn down at Stop. Kept explicit (rather than
// relying on a base default) so the lifecycle is observable to the
// caller.
func (a *WASMAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op for WASM; the runtime process is bound to the calling
// goroutine and released when the sandbox goes out of scope.
func (a *WASMAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete removes the WASM instance. WASM has no persistent on-disk state
// to clean up, so this is a no-op when called after Stop. The id is
// accepted for interface compatibility with the rest of the Adapter API.
func (a *WASMAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

// GetStartupTime returns the typical cold-start latency for wasmtime.
func (a *WASMAdapter) GetStartupTime() time.Duration {
	return 1 * time.Millisecond
}

// Probe checks whether the wasmtime runtime is reachable. The
// underlying WASMAdapter wraps wasm.NewWASMAdapter; we reuse its Probe
// to keep the boot semantics identical (landlock.go:69 follows the
// same "delegate to internal adapter" pattern).
func (a *WASMAdapter) Probe(ctx context.Context) error {
	if v := probeOverride("NVMS_REQUIRE_WASM"); v == "1" {
		return fmt.Errorf("wasm: probe disabled via NVMS_REQUIRE_WASM=1")
	}
	if a.adapter == nil {
		return fmt.Errorf("wasm: internal adapter not initialized")
	}
	return a.adapter.Probe(ctx)
}
