// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — youki.go is the Tier 17 adapter wrapping youki, a
// Rust-based OCI runtime (low-level equivalent of runc, implemented in
// Rust).
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// YoukiAdapter is the Tier 17 youki (Rust OCI runtime) adapter.
// Startup: ~30ms (slightly faster than runc due to no GC pauses), Memory:
// ~30MB, Security: medium (same kernel-shared model as runc/podman but
// written in a memory-safe language).
type YoukiAdapter struct {
	*baseAdapter
	binary string
}

// NewYoukiAdapter creates a new Tier 17 youki adapter.
func NewYoukiAdapter() *YoukiAdapter {
	return &YoukiAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "youki",
			Number:      17,
			DisplayName: "youki (Rust OCI runtime)",
			Description: "youki: Rust-based OCI runtime, drop-in runc replacement (~30ms, ~30MB)",
			StartupMS:   30,
			MemoryMB:    30,
			Security:    "medium",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool", "cli"},
		}},
		binary: "youki",
	}
}

// Deploy creates a youki-managed container descriptor.
func (a *YoukiAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	if config.Image == "" {
		return nil, fmt.Errorf("youki: image is required in SandboxConfig")
	}
	return newSandbox("youki", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start launches the container via `youki run`.
func (a *YoukiAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "run", "--bundle", "/run/youki/"+id, id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("youki start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop sends the kill signal to the container via `youki kill`.
func (a *YoukiAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "kill", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("youki kill %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Delete removes the container via `youki delete`.
func (a *YoukiAdapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "delete", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("youki delete %s: %w: %s", id, err, string(out))
	}
	return nil
}

// GetStartupTime returns the typical youki cold-start latency.
func (a *YoukiAdapter) GetStartupTime() time.Duration { return 30 * time.Millisecond }

// Probe verifies the youki binary is on $PATH.
func (a *YoukiAdapter) Probe(_ context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("youki: binary %q not found: %w", a.binary, err)
	}
	return nil
}
