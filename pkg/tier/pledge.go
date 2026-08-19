// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — pledge.go is the Tier 25 adapter for OpenBSD's pledge(2)
// syscall filter. pledge restricts a process to a small set of
// capability groups and is the canonical OpenBSD sandboxing primitive.
//go:build openbsd

package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// PledgeAdapter is the Tier 25 OpenBSD pledge adapter.
// Startup: ~1ms (a single syscall), Memory: ~1MB,
// Security: low (no filesystem isolation; only syscall-level filter).
type PledgeAdapter struct {
	*baseAdapter
	pledgePath string
}

// NewPledgeAdapter creates a new Tier 25 pledge adapter.
func NewPledgeAdapter() *PledgeAdapter {
	return &PledgeAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "pledge",
			Number:      25,
			DisplayName: "OpenBSD pledge",
			Description: "OpenBSD pledge(2): syscall capability groups (~1ms, ~1MB)",
			StartupMS:   1,
			MemoryMB:    1,
			Security:    "low",
			Platforms:   []string{"openbsd"},
			Workloads:   []string{"tool", "cli"},
		}},
		pledgePath: "/usr/bin/pledge",
	}
}

// Deploy creates an OpenBSD pledge-bounded sandbox record. The actual
// pledge(2) call is made when the wrapped binary is execve(2)'d.
func (a *PledgeAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("pledge", config.Name, domain.SandboxTypeProcess, "", &config), nil
}

// Start is a no-op: pledge is applied at execve time by the wrapped
// command. The pledge(1) helper exists for testing.
func (a *PledgeAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop terminates the wrapped process via pkill on the pledged command.
func (a *PledgeAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", id)
	return cmd.Run()
}

// Delete is a no-op: pledge is in-process and immutable.
func (a *PledgeAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical pledge activation latency.
func (a *PledgeAdapter) GetStartupTime() time.Duration { return 1 * time.Millisecond }

// Probe verifies the pledge(1) binary is on $PATH. Note: this adapter is
// gated to openbsd via the `//go:build openbsd` constraint above; on
// other platforms the file is excluded from the build entirely. On Linux
// and macOS pledge may exist as a compatible shim, but we still gate to
// the native platform.
func (a *PledgeAdapter) Probe(_ context.Context) error {
	if _, err := os.Stat(a.pledgePath); err != nil {
		return fmt.Errorf("pledge: %s not found (need OpenBSD): %w", a.pledgePath, err)
	}
	return nil
}

// registerPledge wires up the OpenBSD pledge adapter. The corresponding
// pledge_other.go stub provides a no-op on non-openbsd platforms.
func registerPledge(r *Registry) {
	r.MustRegister("pledge", NewPledgeAdapter())
}
