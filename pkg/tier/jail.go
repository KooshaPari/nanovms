// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — jail.go is the Tier 24 adapter for FreeBSD jails. Jails
// are the FreeBSD kernel primitive for filesystem + network + user
// namespace isolation, conceptually similar to Linux containers.
//go:build freebsd

package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// JailAdapter is the Tier 24 FreeBSD jail adapter.
// Startup: ~50ms (jail(2) syscall + config load), Memory: ~16MB,
// Security: medium (kernel-level namespace isolation).
type JailAdapter struct {
	*baseAdapter
	binary string
}

// NewJailAdapter creates a new Tier 24 FreeBSD jail adapter.
func NewJailAdapter() *JailAdapter {
	return &JailAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "jail",
			Number:      24,
			DisplayName: "FreeBSD jail",
			Description: "FreeBSD jail: kernel-level filesystem + network + user isolation (~50ms)",
			StartupMS:   50,
			MemoryMB:    16,
			Security:    "medium",
			Platforms:   []string{"freebsd"},
			Workloads:   []string{"code", "tool", "cli"},
		}},
		binary: "jail",
	}
}

// Deploy creates a FreeBSD jail descriptor.
func (a *JailAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("jail", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start brings the jail up via `jail -c`.
func (a *JailAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "-c", "name="+id, "path=/usr/jail/"+id, "command=/bin/sh")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jail start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop tears the jail down via `jail -r`.
func (a *JailAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "-r", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jail stop %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Delete is a no-op: jails are released via `jail -r` (Stop).
func (a *JailAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical jail cold-start latency.
func (a *JailAdapter) GetStartupTime() time.Duration { return 50 * time.Millisecond }

// Probe verifies the jail(8) binary is on $PATH. Note: this adapter is
// gated to freebsd via the `//go:build freebsd` constraint above; on
// other platforms the file is excluded from the build entirely.
func (a *JailAdapter) Probe(_ context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("jail: binary %q not found: %w", a.binary, err)
	}
	return nil
}

// registerJail wires up the FreeBSD jail adapter in the default
// registry. The corresponding jail_other.go stub provides a no-op on
// non-freebsd platforms.
func registerJail(r *Registry) {
	r.MustRegister("jail", NewJailAdapter())
}
