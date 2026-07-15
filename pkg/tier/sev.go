// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — sev.go is the Tier 20 adapter for AMD Secure Encrypted
// Virtualization (SEV / SEV-ES / SEV-SNP), a hardware memory-encryption
// extension that protects guest VM memory from the host.
package tier

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// SEVAdapter is the Tier 20 AMD SEV adapter.
// Startup: ~500ms (SEV launch + qemu boot), Memory: ~256MB,
// Security: untrusted (memory-encrypted VM with attestation).
type SEVAdapter struct {
	*baseAdapter
	device string
}

// NewSEVAdapter creates a new Tier 20 AMD SEV adapter.
func NewSEVAdapter() *SEVAdapter {
	return &SEVAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "sev",
			Number:      20,
			DisplayName: "AMD SEV (memory encryption)",
			Description: "AMD SEV memory-encrypted VM (~500ms startup, ~256MB, attestation)",
			StartupMS:   500,
			MemoryMB:    256,
			Security:    "untrusted",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "tool"},
		}},
		device: "/dev/sev",
	}
}

// Deploy creates an SEV-backed VM descriptor.
func (a *SEVAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("sev", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start is a no-op: SEV attestation and guest launch are handled by qemu
// outside this package.
func (a *SEVAdapter) Start(_ context.Context, _ string) error { return nil }

// Stop tears the SEV guest down; the actual signalling is qemu-driven.
func (a *SEVAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases the SEV descriptor (no persistent state).
func (a *SEVAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical SEV guest boot latency.
func (a *SEVAdapter) GetStartupTime() time.Duration { return 500 * time.Millisecond }

// Probe verifies /dev/sev is readable on a Linux host. The kernel module
// is exposed by the sev driver on supported AMD EPYC CPUs.
func (a *SEVAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("sev: platform %q not supported (linux-only)", runtime.GOOS)
	}
	f, err := os.OpenFile(a.device, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("sev: %s not accessible (need AMD EPYC with SEV + sev driver loaded): %w", a.device, err)
	}
	_ = f.Close()
	return nil
}
