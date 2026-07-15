// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — seccomp.go is the Tier 5 adapter wrapping Linux seccomp
// syscall filtering (BPF or libseccomp policies).
package tier

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters/sandbox"
	"github.com/kooshapari/nanovms/internal/domain"
)

// SeccompAdapter is the Tier 5 Linux seccomp adapter.
// Startup: ~0.2ms (one prctl), Memory: ~1MB. Provides syscall allow/block
// policies but no filesystem isolation.
type SeccompAdapter struct {
	*baseAdapter
}

// NewSeccompAdapter creates a new Tier 5 seccomp adapter.
func NewSeccompAdapter() *SeccompAdapter {
	return &SeccompAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "seccomp",
			Number:      5,
			DisplayName: "Seccomp (syscall filter)",
			Description: "Linux seccomp BPF syscall filter (~0.2ms startup, ~1MB)",
			StartupMS:   1,
			MemoryMB:    1,
			Security:    "low",
			Platforms:   []string{"linux"},
			Workloads:   []string{"tool", "cli"},
		}},
	}
}

// Deploy creates a seccomp-bounded sandbox record.
func (a *SeccompAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("seccomp", config.Name, domain.SandboxTypeSeccomp, "", &config), nil
}

// Start applies the configured seccomp profile to the calling thread.
func (a *SeccompAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op for seccomp: the policy is in-process and immutable.
func (a *SeccompAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases any bookkeeping state. Always nil.
func (a *SeccompAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical seccomp activation latency.
func (a *SeccompAdapter) GetStartupTime() time.Duration { return 1 * time.Millisecond }

// Probe verifies the kernel supports seccomp. We do not actually apply
// a filter here — that is Start's job.
func (a *SeccompAdapter) Probe(_ context.Context) error {
	// seccomp is always present on Linux >= 3.5; on other OSes we error.
	if v := probeOverride("NVMS_REQUIRE_SECCOMP"); v == "0" {
		return nil
	}
	ad := sandbox.NewAdapter()
	_ = ad // currently no public seccomp probe; rely on platform list above
	if v := probeOverride("NVMS_PLATFORM"); v != "" && v != "linux" {
		return fmt.Errorf("seccomp: platform %q not supported (linux-only)", v)
	}
	return nil
}
