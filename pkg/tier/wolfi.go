// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — wolfi.go is the Tier 28 adapter for Wolfi Linux, Chainguard's
// minimal apk-based Linux distribution designed for containers. We probe
// /etc/os-release for the ID=wolfi marker; the actual container is
// typically launched via docker / podman / nerdctl under a Wolfi base image.
package tier

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// WolfiAdapter is the Tier 28 Wolfi Linux adapter.
// Startup: ~50ms (container start on top of a Wolfi image, very small base),
// Memory: ~10MB (no package manager, no shell by default), Security: medium
// (signed repos + minimal surface area).
type WolfiAdapter struct {
	*baseAdapter
	osRelease string
}

// NewWolfiAdapter creates a new Tier 28 Wolfi adapter.
func NewWolfiAdapter() *WolfiAdapter {
	return &WolfiAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "wolfi",
			Number:      28,
			DisplayName: "Wolfi Linux (apk-based)",
			Description: "Wolfi Linux: Chainguard's minimal apk-based distroless (~50ms, ~10MB)",
			StartupMS:   50,
			MemoryMB:    10,
			Security:    "medium",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool", "cli"},
		}},
		osRelease: "/etc/os-release",
	}
}

// Deploy creates a Wolfi-backed container descriptor.
func (a *WolfiAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("wolfi", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start is a no-op: Wolfi is a guest image; the actual container launch
// is driven by docker / podman / nerdctl outside this adapter.
func (a *WolfiAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op: same rationale as Start.
func (a *WolfiAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases the descriptor.
func (a *WolfiAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical Wolfi container start latency.
func (a *WolfiAdapter) GetStartupTime() time.Duration { return 50 * time.Millisecond }

// Probe verifies the host / guest is running Wolfi by parsing
// /etc/os-release. We accept either an exact ID=wolfi match or the
// case-insensitive substring.
func (a *WolfiAdapter) Probe(_ context.Context) error {
	f, err := os.Open(a.osRelease)
	if err != nil {
		return fmt.Errorf("wolfi: %s not readable: %w", a.osRelease, err)
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	contents := string(buf[:n])
	if !strings.Contains(strings.ToLower(contents), "id=wolfi") &&
		!strings.Contains(strings.ToLower(contents), `id_like=`+"wolfi") {
		return fmt.Errorf("wolfi: /etc/os-release does not advertise a Wolfi ID")
	}
	return nil
}
