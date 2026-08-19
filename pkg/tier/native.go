// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — native.go is the Tier 6 adapter for the "native" baseline
// (no isolation; useful as a performance reference and for trusted code).
package tier

import (
	"context"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// NativeAdapter is the Tier 6 "no isolation" baseline.
// Startup: ~0ms, Memory: ~0MB overhead. Runs the workload directly in the
// caller process. Use only for fully trusted code.
type NativeAdapter struct {
	*baseAdapter
}

// NewNativeAdapter creates a new Tier 6 native adapter.
func NewNativeAdapter() *NativeAdapter {
	return &NativeAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "native",
			Number:      6,
			DisplayName: "Native (no isolation)",
			Description: "Run the workload directly in the calling process; zero overhead baseline",
			StartupMS:   0,
			MemoryMB:    0,
			Security:    "low",
			Platforms:   []string{"linux", "macos", "windows"},
			Workloads:   []string{"cli", "tool"},
		}},
	}
}

// Deploy returns a sandbox descriptor for native execution.
func (a *NativeAdapter) Deploy(_ context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	return newSandbox("native", config.Name, domain.SandboxTypeNative, "", &config), nil
}

// Start is a no-op: native execution is implicit.
func (a *NativeAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op: native execution is implicit.
func (a *NativeAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete is a no-op: nothing to clean up.
func (a *NativeAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns zero: native execution has no startup cost.
func (a *NativeAdapter) GetStartupTime() time.Duration { return 0 }

// Probe always succeeds — native execution requires no runtime.
func (a *NativeAdapter) Probe(_ context.Context) error { return nil }
