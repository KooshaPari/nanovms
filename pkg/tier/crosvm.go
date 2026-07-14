// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — crosvm.go is the Tier 15 adapter wrapping crosvm, the
// ChromeOS virtual machine monitor.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// CrosvmAdapter is the Tier 15 ChromeOS crosvm adapter.
// Startup: ~200ms, Memory: ~128MB. crosvm is Linux-only and originally
// targets KVM on x86_64 / aarch64.
type CrosvmAdapter struct {
	*baseAdapter
	binary string
}

// NewCrosvmAdapter creates a new Tier 15 crosvm adapter.
func NewCrosvmAdapter() *CrosvmAdapter {
	return &CrosvmAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "crosvm",
			Number:      15,
			DisplayName: "crosvm (ChromeOS VMM)",
			Description: "ChromeOS crosvm VMM with virtio devices (~200ms startup, ~128MB)",
			StartupMS:   200,
			MemoryMB:    128,
			Security:    "high",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool"},
		}},
		binary: "crosvm",
	}
}

// Deploy creates a crosvm VM descriptor.
func (a *CrosvmAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("crosvm", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start launches the crosvm VM in the background with a writable serial
// socket for control.
func (a *CrosvmAdapter) Start(ctx context.Context, id string) error {
	args := []string{
		"run",
		"-s", "/tmp/" + id + ".serial",
		"--rwdisk", "/tmp/" + id + ".img",
	}
	cmd := exec.CommandContext(ctx, a.binary, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crosvm start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the crosvm VM via pkill.
func (a *CrosvmAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "crosvm.*"+id)
	return cmd.Run()
}

// Delete stops the crosvm VM and releases its state.
func (a *CrosvmAdapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// GetStartupTime returns the typical crosvm boot latency.
func (a *CrosvmAdapter) GetStartupTime() time.Duration { return 200 * time.Millisecond }

// Probe verifies the crosvm binary is on $PATH and we are on Linux.
func (a *CrosvmAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("crosvm: platform %q not supported (linux-only)", runtime.GOOS)
	}
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("crosvm: binary %q not found: %w", a.binary, err)
	}
	return nil
}
