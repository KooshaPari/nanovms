// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — lima.go is the Tier 11 adapter that launches a Linux VM
// via Lima (or colima) on macOS/Windows.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// LimaAdapter is the Tier 11 Linux-on-host VM adapter.
// Startup: ~3000ms, Memory: ~512MB. Wraps `limactl` (or `colima` as a
// fallback) to provide a Linux VM on macOS/Windows hosts.
type LimaAdapter struct {
	*baseAdapter
	binary string
}

// NewLimaAdapter creates a new Tier 11 Lima adapter. The underlying
// binary is resolved lazily by Probe; we default to `limactl` here.
func NewLimaAdapter() *LimaAdapter {
	return &LimaAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "lima",
			Number:      11,
			DisplayName: "Lima (Linux VM)",
			Description: "Lima-managed Linux VM (~3s startup, ~512MB) on macOS/Windows",
			StartupMS:   3000,
			MemoryMB:    512,
			Security:    "high",
			Platforms:   []string{"macos", "windows"},
			Workloads:   []string{"code", "tool", "browser"},
		}},
		binary: "limactl",
	}
}

// resolveBinary prefers limactl and falls back to colima if present.
func (a *LimaAdapter) resolveBinary() (string, error) {
	if p, err := exec.LookPath("limactl"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("colima"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("lima: neither limactl nor colima found on $PATH")
}

// Deploy creates a Lima instance descriptor. The instance is not started.
func (a *LimaAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("lima", config.Name, domain.SandboxTypeVM, domain.VMFlavorLima, &config), nil
}

// Start launches the Lima instance via `limactl start`.
func (a *LimaAdapter) Start(ctx context.Context, id string) error {
	bin, err := a.resolveBinary()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, "start", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lima start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop stops the Lima instance via `limactl stop`.
func (a *LimaAdapter) Stop(ctx context.Context, id string) error {
	bin, err := a.resolveBinary()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, "stop", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lima stop %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Delete removes the Lima instance via `limactl delete -f`.
func (a *LimaAdapter) Delete(ctx context.Context, id string) error {
	bin, err := a.resolveBinary()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, "delete", "-f", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lima delete %s: %w: %s", id, err, string(out))
	}
	return nil
}

// GetStartupTime returns the typical Lima instance start latency.
func (a *LimaAdapter) GetStartupTime() time.Duration { return 3000 * time.Millisecond }

// Probe verifies limactl or colima is on $PATH.
func (a *LimaAdapter) Probe(_ context.Context) error {
	if _, err := a.resolveBinary(); err != nil {
		return err
	}
	return nil
}
