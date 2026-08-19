// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — virtcontainers.go is the Tier 23 adapter for the legacy
// Intel Clear Containers / virtcontainers project (now archived;
// superseded by Kata Containers, but kept here for back-compat with
// deployments still pinned on it).
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// VirtContainersAdapter is the Tier 23 Intel Clear Containers /
// virtcontainers adapter.
// Startup: ~200ms (qemu-lite boot, similar to Firecracker), Memory:
// ~128MB, Security: medium (clear-container user-space kernel, legacy).
type VirtContainersAdapter struct {
	*baseAdapter
	binary string
}

// NewVirtContainersAdapter creates a new Tier 23 virtcontainers adapter.
func NewVirtContainersAdapter() *VirtContainersAdapter {
	return &VirtContainersAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "virtcontainers",
			Number:      23,
			DisplayName: "Intel Clear Containers (virtcontainers)",
			Description: "virtcontainers: legacy Intel Clear Containers (~200ms, ~128MB)",
			StartupMS:   200,
			MemoryMB:    128,
			Security:    "medium",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool"},
		}},
		binary: "virtcontainers",
	}
}

// Deploy creates a virtcontainers-managed pod descriptor.
func (a *VirtContainersAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("virtcontainers", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start launches the legacy Clear Containers runtime.
func (a *VirtContainersAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "run", "--id", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virtcontainers start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the Clear Container via pkill on the runtime id.
func (a *VirtContainersAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "virtcontainers.*"+id)
	return cmd.Run()
}

// Delete releases the descriptor. virtcontainers is process-scoped.
func (a *VirtContainersAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical Clear Containers boot latency.
func (a *VirtContainersAdapter) GetStartupTime() time.Duration { return 200 * time.Millisecond }

// Probe verifies the virtcontainers runtime binary is on $PATH. Note:
// virtcontainers was archived upstream in 2019 in favor of Kata
// Containers; this adapter is kept for legacy consumers.
func (a *VirtContainersAdapter) Probe(_ context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("virtcontainers: binary %q not found (archived in 2019; use Kata): %w", a.binary, err)
	}
	return nil
}
