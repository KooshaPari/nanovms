// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — landlock.go is the Tier 4 adapter wrapping Linux
// Landlock path-based sandboxing (kernel >= 5.13).
package tier

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters/sandbox"
	"github.com/kooshapari/nanovms/internal/domain"
)

// LandlockAdapter is the Tier 4 Linux Landlock adapter.
// Startup: ~1ms (just a syscall pair), Memory: ~2MB, no kernel module
// required beyond the Landlock LSM that ships in mainline since 5.13.
type LandlockAdapter struct {
	*baseAdapter
}

// NewLandlockAdapter creates a new Tier 4 Landlock adapter.
func NewLandlockAdapter() *LandlockAdapter {
	return &LandlockAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "landlock",
			Number:      4,
			DisplayName: "Landlock (path-based)",
			Description: "Linux Landlock LSM providing path-based filesystem isolation (~1ms, ~2MB)",
			StartupMS:   1,
			MemoryMB:    2,
			Security:    "low",
			Platforms:   []string{"linux"},
			Workloads:   []string{"tool", "cli"},
		}},
	}
}

// Deploy creates a Landlock-backed sandbox record. Actual restriction is
// applied in Start via the underlying sandbox.landlockAdapter.
func (a *LandlockAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("landlock", config.Name, domain.SandboxTypeLandlock, "", &config), nil
}

// Start applies the Landlock ruleset to the current thread using real
// kernel syscalls via the sandbox.landlockRestrictSelf entry point.
func (a *LandlockAdapter) Start(_ context.Context, _ string) error {
	// Build the default read-only ruleset (deny write to everything except /tmp)
	// and apply it to the calling thread. The ruleset fd is closed internally.
	rulesetFd, err := sandbox.BuildLandlockRulesetDefault()
	if err != nil {
		return fmt.Errorf("landlock: failed to build ruleset: %w", err)
	}

	// Apply restriction to calling thread
	if err := sandbox.LandlockRestrictSelf(rulesetFd); err != nil {
		return fmt.Errorf("landlock: failed to restrict self: %w", err)
	}
	return nil
}

// Stop is a no-op for Landlock: the ruleset is enforced in-process.
func (a *LandlockAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases any bookkeeping state. Always nil.
func (a *LandlockAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical Landlock activation latency.
func (a *LandlockAdapter) GetStartupTime() time.Duration { return 1 * time.Millisecond }

// Probe checks kernel Landlock support via the internal helper.
func (a *LandlockAdapter) Probe(_ context.Context) error {
	// Real probe would consult kernelSupportsLandlockWrapper; we
	// accept the env var NVMS_REQUIRE_LANDLOCK as an override hook.
	if v := probeOverride("NVMS_REQUIRE_LANDLOCK"); v == "0" {
		return nil
	}
	// The internal sandbox.Adapter exposes ListRuntimes which we can
	// reuse to check kernel support without re-implementing the
	// syscall probe here.
	ad := sandbox.NewAdapter()
	rts, err := ad.ListRuntimes(context.Background())
	if err != nil {
		return fmt.Errorf("landlock: probe failed: %w", err)
	}
	for _, r := range rts {
		if r.Type == domain.SandboxTypeLandlock {
			return nil
		}
	}
	return fmt.Errorf("landlock: kernel does not advertise Landlock support (need Linux >= 5.13)")
}
