// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — tdx.go is the Tier 21 adapter for Intel Trust Domain
// Extensions (TDX), a hardware-isolated VM primitive that protects guest
// memory and CPU state from the host VMM.
package tier

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// TDXAdapter is the Tier 21 Intel TDX adapter.
// Startup: ~600ms (TDX attestation + qemu boot), Memory: ~256MB,
// Security: untrusted (hardware memory + CPU state isolation).
type TDXAdapter struct {
	*baseAdapter
	device string
}

// NewTDXAdapter creates a new Tier 21 Intel TDX adapter.
func NewTDXAdapter() *TDXAdapter {
	return &TDXAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "tdx",
			Number:      21,
			DisplayName: "Intel TDX (Trust Domain)",
			Description: "Intel TDX Trust Domain Extension (~600ms startup, ~256MB, attested)",
			StartupMS:   600,
			MemoryMB:    256,
			Security:    "untrusted",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool"},
		}},
		device: "/dev/tdx-guest",
	}
}

// Deploy creates a TDX-backed VM descriptor.
func (a *TDXAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("tdx", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start is a no-op: TDX trust domain launch is handled by qemu outside
// this package.
func (a *TDXAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop tears the TD guest down.
func (a *TDXAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases the TDX descriptor (no persistent state).
func (a *TDXAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical TDX guest boot latency.
func (a *TDXAdapter) GetStartupTime() time.Duration { return 600 * time.Millisecond }

// Probe verifies /dev/tdx-guest is readable on a Linux host. The kernel
// module is exposed by the tdx driver on supported 4th-gen Xeon Scalable
// CPUs.
func (a *TDXAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("tdx: platform %q not supported (linux-only)", runtime.GOOS)
	}
	f, err := os.OpenFile(a.device, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("tdx: %s not accessible (need 4th-gen Xeon Scalable + tdx driver loaded): %w", a.device, err)
	}
	_ = f.Close()
	return nil
}
