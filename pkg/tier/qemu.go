// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — qemu.go is the Tier 13 full-system emulator adapter that
// drives the `qemu-system-*` CLI family.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// QEMUAdapter is the Tier 13 full-system emulator adapter.
// Startup: ~2000ms, Memory: ~256MB. The most portable VM tier: runs on
// Linux, macOS, and Windows; supports x86_64, aarch64, riscv64, ppc, etc.
type QEMUAdapter struct {
	*baseAdapter
	binary string
}

// NewQEMUAdapter creates a new Tier 13 QEMU adapter.
func NewQEMUAdapter() *QEMUAdapter {
	return &QEMUAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "qemu",
			Number:      13,
			DisplayName: "QEMU (full system emulator)",
			Description: "QEMU full-system emulation (~2s startup, ~256MB); supports x86_64/aarch64/riscv64",
			StartupMS:   2000,
			MemoryMB:    256,
			Security:    "high",
			Platforms:   []string{"linux", "macos", "windows"},
			Workloads:   []string{"code", "browser", "tool"},
		}},
		binary: "qemu-system-x86_64",
	}
}

// resolveBinary picks the first qemu-system-* binary on $PATH.
func (a *QEMUAdapter) resolveBinary() (string, error) {
	if p, err := exec.LookPath("qemu-system-x86_64"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("qemu-system-aarch64"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("qemu-kvm"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("qemu: no qemu-system-* binary found on $PATH")
}

// Deploy creates a QEMU VM descriptor.
func (a *QEMUAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("qemu", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start launches the QEMU VM with the minimum flags required to background.
func (a *QEMUAdapter) Start(ctx context.Context, id string) error {
	bin, err := a.resolveBinary()
	if err != nil {
		return err
	}
	args := []string{
		"-name", id,
		"-daemonize",
		"-nographic",
		"-m", "512",
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the QEMU VM via pkill.
func (a *QEMUAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "qemu.*-name "+id)
	return cmd.Run()
}

// Delete stops the QEMU VM and releases its state.
func (a *QEMUAdapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// GetStartupTime returns the typical QEMU boot latency.
func (a *QEMUAdapter) GetStartupTime() time.Duration { return 2000 * time.Millisecond }

// Probe verifies a qemu-system-* binary is on $PATH.
func (a *QEMUAdapter) Probe(_ context.Context) error {
	if _, err := a.resolveBinary(); err != nil {
		return err
	}
	return nil
}
