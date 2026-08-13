// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — podman.go is the Tier 8 rootless OCI container adapter
// that drives the `podman` CLI.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// PodmanAdapter is the Tier 8 rootless OCI container adapter.
// Startup: ~700ms, Memory: ~50MB overhead. Works rootless on Linux
// without a daemon.
type PodmanAdapter struct {
	*baseAdapter
	binary string
}

// NewPodmanAdapter creates a new Tier 8 Podman adapter.
func NewPodmanAdapter() *PodmanAdapter {
	return &PodmanAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "podman",
			Number:      8,
			DisplayName: "Podman (rootless OCI)",
			Description: "Podman rootless OCI containers (~700ms startup, ~50MB)",
			StartupMS:   700,
			MemoryMB:    50,
			Security:    "medium",
			Platforms:   []string{"linux", "macos", "windows"},
			Workloads:   []string{"code", "tool", "cli"},
		}},
		binary: "podman",
	}
}

// Deploy creates a Podman container (not started).
func (a *PodmanAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	if config.Image == "" {
		return nil, fmt.Errorf("podman: image is required in SandboxConfig")
	}
	return newSandbox("podman", config.Name, domain.SandboxTypeContainer, "", &config), nil
}

// Start launches the container via `podman start`.
func (a *PodmanAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "start", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("podman start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the container via `podman stop`.
func (a *PodmanAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "stop", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("podman stop %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Delete removes the container via `podman rm -f`.
func (a *PodmanAdapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "rm", "-f", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("podman rm %s: %w: %s", id, err, string(out))
	}
	return nil
}

// GetStartupTime returns the typical Podman container start latency.
func (a *PodmanAdapter) GetStartupTime() time.Duration { return 700 * time.Millisecond }

// Probe verifies the podman CLI can inspect its local lifecycle store. Avoid
// `podman version --format`: rootless/WSL connection discovery may hang even
// when local create/run operations are healthy.
func (a *PodmanAdapter) Probe(ctx context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("podman: binary %q not found: %w", a.binary, err)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, a.binary, "ps", "--all", "--noheading")
	if out, err := cmd.CombinedOutput(); err != nil {
		if probeCtx.Err() != nil {
			return fmt.Errorf("podman: probe timed out after 10s")
		}
		return fmt.Errorf("podman: not functional: %w (%s)", err, string(out))
	}
	return nil
}
