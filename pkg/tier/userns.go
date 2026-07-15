// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — userns.go is the Tier 30 adapter for Linux user
// namespaces. Unprivileged userns lets a non-root process gain a fake
// root inside its own namespace; we detect this by reading
// /proc/self/uid_map and verifying the kernel supports userns.
package tier

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// UserNSAdapter is the Tier 30 Linux user-namespace adapter.
// Startup: ~1ms (a single clone(2) call), Memory: ~1MB,
// Security: low (no kernel backing; relies on the rest of the kernel for
// syscall isolation; the runtime is the least-privileged tier).
type UserNSAdapter struct {
	*baseAdapter
	uidMapPath string
}

// NewUserNSAdapter creates a new Tier 30 user-namespace adapter.
func NewUserNSAdapter() *UserNSAdapter {
	return &UserNSAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "userns",
			Number:      30,
			DisplayName: "Linux user namespaces",
			Description: "Linux user namespaces (unprivileged uid/gid mapping, ~1ms startup)",
			StartupMS:   1,
			MemoryMB:    1,
			Security:    "low",
			Platforms:   []string{"linux"},
			Workloads:   []string{"tool", "cli"},
		}},
		uidMapPath: "/proc/self/uid_map",
	}
}

// Deploy creates a user-namespace scoped descriptor.
func (a *UserNSAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("userns", config.Name, domain.SandboxTypeProcess, "", &config), nil
}

// Start is a no-op: the unshare(2) / clone(2) call is made on the wrapping
// process when it enters the userns; this adapter only tracks lifecycle.
func (a *UserNSAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op: userns scopes are released when the wrapped process
// exits; there is no external supervisor to signal.
func (a *UserNSAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases the descriptor.
func (a *UserNSAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical userns activation latency.
func (a *UserNSAdapter) GetStartupTime() time.Duration { return 1 * time.Millisecond }

// Probe verifies /proc/self/uid_map is readable on a Linux host. The
// kernel always exposes this procfs node, but on hosts with userns
// disabled (kernel.unprivileged_userns_clone=0) the file may be empty;
// we treat an open-able file with a non-empty body as success and an
// empty body as "userns disabled".
func (a *UserNSAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("userns: platform %q not supported (linux-only)", runtime.GOOS)
	}
	f, err := os.Open(a.uidMapPath)
	if err != nil {
		return fmt.Errorf("userns: %s not readable: %w", a.uidMapPath, err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("userns: %s stat failed: %w", a.uidMapPath, err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("userns: %s is empty (kernel.unprivileged_userns_clone disabled)", a.uidMapPath)
	}
	return nil
}
