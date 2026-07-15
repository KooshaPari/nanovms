// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — sandboxexec.go is the Tier 26 adapter for macOS
// sandbox-exec, Apple's built-in sandbox profile executor introduced in
// 10.5. We drive it via sandbox-exec(1) with a minimal SBPL profile.
//go:build darwin

package tier

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// SandboxExecAdapter is the Tier 26 macOS sandbox-exec adapter.
// Startup: ~5ms (process re-exec with profile), Memory: ~2MB,
// Security: low (SBPL profile-based).
type SandboxExecAdapter struct {
	*baseAdapter
	binary string
}

// NewSandboxExecAdapter creates a new Tier 26 sandbox-exec adapter.
func NewSandboxExecAdapter() *SandboxExecAdapter {
	return &SandboxExecAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "sandboxexec",
			Number:      26,
			DisplayName: "macOS sandbox-exec",
			Description: "macOS sandbox-exec with SBPL profile (~5ms startup, ~2MB)",
			StartupMS:   5,
			MemoryMB:    2,
			Security:    "low",
			Platforms:   []string{"macos"},
			Workloads:   []string{"tool", "cli"},
		}},
		binary: "/usr/bin/sandbox-exec",
	}
}

// Deploy creates a sandbox-exec scoped descriptor.
func (a *SandboxExecAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("sandboxexec", config.Name, domain.SandboxTypeProcess, "", &config), nil
}

// Start execs the configured command inside the SBPL profile.
func (a *SandboxExecAdapter) Start(ctx context.Context, id string) error {
	// The minimal SBPL profile below denies everything except basic
	// process / stdio operations. A real workload would supply its
	// own profile via config.
	profile := "(version 1)(deny default)(allow process-exec)(allow stdio)"
	cmd := exec.CommandContext(ctx, a.binary, "-p", profile, "/bin/sh", "-c", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sandbox-exec start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the inner command via pkill.
func (a *SandboxExecAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "sandbox-exec.*"+id)
	return cmd.Run()
}

// Delete releases any bookkeeping state.
func (a *SandboxExecAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical sandbox-exec activation latency.
func (a *SandboxExecAdapter) GetStartupTime() time.Duration { return 5 * time.Millisecond }

// Probe verifies /usr/bin/sandbox-exec exists on the host. Note: this
// adapter is gated to darwin via the `//go:build darwin` constraint
// above; on other platforms the file is excluded from the build entirely.
func (a *SandboxExecAdapter) Probe(_ context.Context) error {
	if _, err := os.Stat(a.binary); err != nil {
		return fmt.Errorf("sandboxexec: %s not found: %w", a.binary, err)
	}
	return nil
}

// registerSandboxExec wires up the macOS sandbox-exec adapter. The
// corresponding sandboxexec_other.go stub provides a no-op on non-
// darwin platforms.
func registerSandboxExec(r *Registry) {
	r.MustRegister("sandboxexec", NewSandboxExecAdapter())
}
