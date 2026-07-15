// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — systemd-nspawn.go is the Tier 18 adapter wrapping the
// systemd-nspawn container engine. nspawn is built into systemd and uses
// Linux namespaces + cgroups for OCI-like isolation on systemd hosts.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// SystemdNspawnAdapter is the Tier 18 systemd-nspawn adapter.
// Startup: ~100ms (image unpack + namespace setup), Memory: ~40MB,
// Security: medium (kernel namespaces; no VM).
type SystemdNspawnAdapter struct {
	*baseAdapter
	binary string
}

// NewSystemdNspawnAdapter creates a new Tier 18 systemd-nspawn adapter.
func NewSystemdNspawnAdapter() *SystemdNspawnAdapter {
	return &SystemdNspawnAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "systemdnspawn",
			Number:      18,
			DisplayName: "systemd-nspawn",
			Description: "systemd-nspawn namespace container engine (~100ms, ~40MB)",
			StartupMS:   100,
			MemoryMB:    40,
			Security:    "medium",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool", "cli"},
		}},
		binary: "systemd-nspawn",
	}
}

// Deploy creates a systemd-nspawn machine descriptor. The machine is
// referenced by name and lazily booted on Start.
func (a *SystemdNspawnAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("systemdnspawn", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start boots the nspawn machine via `systemd-nspawn --boot`.
func (a *SystemdNspawnAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "--boot", "--machine", id, "--directory", "/var/lib/machines/"+id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemd-nspawn start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the machine via `machinectl poweroff`.
func (a *SystemdNspawnAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "machinectl", "poweroff", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemd-nspawn stop %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Delete is a no-op: machine directories are managed by machinectl out of
// band.
func (a *SystemdNspawnAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical systemd-nspawn boot latency.
func (a *SystemdNspawnAdapter) GetStartupTime() time.Duration { return 100 * time.Millisecond }

// Probe verifies the systemd-nspawn binary is on $PATH.
func (a *SystemdNspawnAdapter) Probe(_ context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("systemd-nspawn: binary %q not found: %w", a.binary, err)
	}
	return nil
}
