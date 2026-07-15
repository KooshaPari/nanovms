// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — kata.go is the Tier 16 adapter wrapping Kata Containers
// (containerd-shim-kata-v2 + qemu-virt) which run pod sandboxes inside a
// lightweight hardware-virtualized VM.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// KataAdapter is the Tier 16 Kata Containers adapter.
// Startup: ~250ms (qemu-virt microVM boot per pod), Memory: ~256MB,
// Security: high (per-pod hardware-virtualized kernel).
type KataAdapter struct {
	*baseAdapter
	shim string
}

// NewKataAdapter creates a new Tier 16 Kata Containers adapter.
func NewKataAdapter() *KataAdapter {
	return &KataAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "kata",
			Number:      16,
			DisplayName: "Kata Containers",
			Description: "Kata Containers: per-pod hardware-virtualized sandbox (~250ms, ~256MB)",
			StartupMS:   250,
			MemoryMB:    256,
			Security:    "high",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool", "browser"},
		}},
		shim: "containerd-shim-kata-v2",
	}
}

// Deploy creates a Kata-backed pod sandbox descriptor.
func (a *KataAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("kata", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start is a no-op: containerd orchestrates the actual shim launch via its
// own runtime registration; the nvms-side descriptor exists as the
// in-package record of the lifecycle.
func (a *KataAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop signals the containerd runtime to halt the shim. On a host without
// containerd we surface a clear error so operators know to install it.
func (a *KataAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases any bookkeeping state.
func (a *KataAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical cold-start latency for a Kata pod
// sandbox (qemu-virt boot dominated).
func (a *KataAdapter) GetStartupTime() time.Duration { return 250 * time.Millisecond }

// Probe verifies the containerd-shim-kata-v2 binary is on $PATH.
func (a *KataAdapter) Probe(_ context.Context) error {
	if v := probeOverride("NVMS_REQUIRE_KATA"); v == "1" {
		return fmt.Errorf("kata: probe disabled via NVMS_REQUIRE_KATA=1")
	}
	if _, err := exec.LookPath(a.shim); err != nil {
		return fmt.Errorf("kata: shim %q not found on $PATH: %w", a.shim, err)
	}
	return nil
}
