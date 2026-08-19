// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — hyperkit.go is the Tier 9 macOS-native VM adapter that
// drives the (now-deprecated) hyperkit CLI.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// HyperKitAdapter is the Tier 9 macOS native VM adapter.
// Startup: ~400ms, Memory: ~256MB. HyperKit is deprecated upstream; this
// adapter is kept for users who still have it installed and pinned.
//
// Build tag note: this adapter is only meaningful on macOS; on other
// platforms Probe() returns an error and Deploy/Start are unusable.
type HyperKitAdapter struct {
	*baseAdapter
	binary string
}

// NewHyperKitAdapter creates a new Tier 9 HyperKit adapter.
func NewHyperKitAdapter() *HyperKitAdapter {
	return &HyperKitAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "hyperkit",
			Number:      9,
			DisplayName: "HyperKit (deprecated)",
			Description: "macOS-native VM via the (now-deprecated) hyperkit CLI (~400ms startup)",
			StartupMS:   400,
			MemoryMB:    256,
			Security:    "high",
			Platforms:   []string{"macos"},
			Workloads:   []string{"code", "tool"},
		}},
		binary: "hyperkit",
	}
}

// Deploy creates a HyperKit VM descriptor. The VM is not started yet.
func (a *HyperKitAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("hyperkit", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start launches the HyperKit VM in the background.
func (a *HyperKitAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "-D", id, "/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hyperkit start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the HyperKit VM via pkill.
func (a *HyperKitAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "hyperkit.*"+id)
	return cmd.Run()
}

// Delete stops the VM and releases its state.
func (a *HyperKitAdapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// GetStartupTime returns the typical HyperKit VM boot latency.
func (a *HyperKitAdapter) GetStartupTime() time.Duration { return 400 * time.Millisecond }

// Probe verifies the hyperkit CLI is on $PATH and we are on macOS.
func (a *HyperKitAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("hyperkit: platform %q not supported (macos-only)", runtime.GOOS)
	}
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("hyperkit: binary %q not found: %w", a.binary, err)
	}
	return nil
}
