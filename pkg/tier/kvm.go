// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — kvm.go is the Tier 12 Linux KVM adapter providing direct
// access to /dev/kvm for user-mode QEMU/Firecracker-style VMs.
package tier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// KVMAdapter is the Tier 12 Linux KVM hypervisor adapter.
// Startup: ~150ms, Memory: ~128MB. Provides direct /dev/kvm access — the
// same underlying primitive Firecracker and QEMU use, but exposed as a
// stand-alone tier for callers that want to drive QEMU directly.
type KVMAdapter struct {
	*baseAdapter
}

// NewKVMAdapter creates a new Tier 12 KVM adapter.
func NewKVMAdapter() *KVMAdapter {
	return &KVMAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "kvm",
			Number:      12,
			DisplayName: "KVM (Linux hypervisor)",
			Description: "Linux KVM (/dev/kvm) direct hypervisor access (~150ms startup, ~128MB)",
			StartupMS:   150,
			MemoryMB:    128,
			Security:    "high",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "browser", "tool"},
		}},
	}
}

// Deploy rejects guest creation until a KVM VM implementation is connected.
// Access to /dev/kvm alone does not create a guest or establish isolation.
func (a *KVMAdapter) Deploy(_ context.Context, _ domain.SandboxConfig) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("kvm: guest deployment is not implemented: %w", errors.ErrUnsupported)
}

// Start must not acknowledge a guest boot that this adapter cannot perform.
func (a *KVMAdapter) Start(_ context.Context, _ string) error {
	return fmt.Errorf("kvm: guest startup is not implemented: %w", errors.ErrUnsupported)
}

// Stop tears the KVM guest down. The actual signalling is hypervisor-
// specific; we expose a no-op stub that downstream code overrides.
func (a *KVMAdapter) Stop(_ context.Context, _ string) error { return nil }

// Delete releases the KVM VM descriptor.
func (a *KVMAdapter) Delete(_ context.Context, _ string) error { return nil }

// GetStartupTime returns the typical KVM boot latency.
func (a *KVMAdapter) GetStartupTime() time.Duration { return 150 * time.Millisecond }

// Probe verifies /dev/kvm is readable on a Linux host.
func (a *KVMAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("kvm: platform %q not supported (linux-only)", runtime.GOOS)
	}
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("kvm: /dev/kvm not accessible: %w", err)
	}
	_ = f.Close()
	return nil
}
