// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier provides public tier adapters for NVMS isolation levels.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// GVisorAdapter is the Tier2 gVisor adapter for semi-trusted workloads.
// Startup: ~90ms, Memory: ~50MB, CPU overhead: ~5%
type GVisorAdapter struct {
	*baseAdapter
	runtime string
}

// NewGVisorAdapter creates a new Tier2 gVisor adapter.
func NewGVisorAdapter() *GVisorAdapter {
	return &GVisorAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "gvisor",
			Number:      2,
			DisplayName: "gVisor runsc",
			Description: "User-space kernel intercepting syscalls (~90ms startup, ~50MB)",
			StartupMS:   90,
			MemoryMB:    50,
			Security:    "medium",
			Platforms:   []string{"linux", "macos"},
			Workloads:   []string{"browser", "code", "tool"},
		}},
		runtime: "runsc",
	}
}

// Deploy deploys a gVisor sandbox workload.
func (a *GVisorAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if _, err := exec.LookPath(a.runtime); err != nil {
		return nil, fmt.Errorf("gVisor runtime (%s) not found: %w", a.runtime, err)
	}

	sandbox := &domain.Sandbox{
		ID:     fmt.Sprintf("gvisor-%s", domain.GenerateID()),
		Name:   config.Name,
		Status: domain.SandboxStatusRunning,
		Type:   domain.SandboxTypeGVisor,
		Config: &config,
	}
	return sandbox, nil
}

// Start launches a gVisor sandbox. runsc is the standard entry point
// and mirrors the firecracker Start pattern at firecracker.go:55.
func (a *GVisorAdapter) Start(ctx context.Context, id string) error {
	path, err := exec.LookPath(a.runtime)
	if err != nil {
		return fmt.Errorf("gVisor runtime (%s) not found: %w", a.runtime, err)
	}
	cmd := exec.CommandContext(ctx, path, "run", "--id", id)
	return cmd.Start()
}

// Stop terminates a gVisor sandbox. Mirrors firecracker.Stop at
// firecracker.go:69 in spirit (pkill -f) but uses the runsc binary
// signal surface.
func (a *GVisorAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", fmt.Sprintf("%s.*%s", a.runtime, id))
	return cmd.Run()
}

// Delete removes a gVisor sandbox. gVisor is a process-scoped sandbox so
// cleanup is delegated to Stop; this method exists to satisfy the
// Adapter interface for the registry.
func (a *GVisorAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

// GetStartupTime returns the typical cold-start latency for runsc.
func (a *GVisorAdapter) GetStartupTime() time.Duration {
	return 90 * time.Millisecond
}

// Probe checks whether the runsc binary is present on the host. The
// firecracker adapter (firecracker.go:98), qemu adapter (qemu.go:43)
// and docker adapter (docker.go:85) all follow the same exec.LookPath
// pattern for their primary binary.
func (a *GVisorAdapter) Probe(_ context.Context) error {
	if v := probeOverride("NVMS_REQUIRE_GVISOR"); v == "1" {
		return fmt.Errorf("gvisor: probe disabled via NVMS_REQUIRE_GVISOR=1")
	}
	if _, err := exec.LookPath(a.runtime); err != nil {
		return fmt.Errorf("gvisor: %s binary not found: %w", a.runtime, err)
	}
	return nil
}
