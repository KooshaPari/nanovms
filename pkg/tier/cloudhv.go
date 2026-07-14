// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — cloudhv.go is the Tier 14 adapter wrapping Cloud
// Hypervisor (https://github.com/cloud-hypervisor/cloud-hypervisor), a
// Rust VMM with KVM/macOS HVF backends.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// CloudHypervisorAdapter is the Tier 14 Cloud Hypervisor VMM adapter.
// Startup: ~180ms, Memory: ~128MB. Cloud Hypervisor is Linux-only; on
// macOS it can use the HVF backend but is experimental.
type CloudHypervisorAdapter struct {
	*baseAdapter
	binary string
}

// NewCloudHypervisorAdapter creates a new Tier 14 Cloud Hypervisor adapter.
func NewCloudHypervisorAdapter() *CloudHypervisorAdapter {
	return &CloudHypervisorAdapter{
		baseAdapter: &baseAdapter{info: TierInfo{
			Name:        "cloudhv",
			Number:      14,
			DisplayName: "Cloud Hypervisor (Rust VMM)",
			Description: "Cloud Hypervisor Rust VMM with KVM backend (~180ms startup, ~128MB)",
			StartupMS:   180,
			MemoryMB:    128,
			Security:    "high",
			Platforms:   []string{"linux"},
			Workloads:   []string{"code", "browser", "tool"},
		}},
		binary: "cloud-hypervisor",
	}
}

// Deploy creates a Cloud Hypervisor VM descriptor.
func (a *CloudHypervisorAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	return newSandbox("cloudhv", config.Name, domain.SandboxTypeVM, domain.VMFlavorNative, &config), nil
}

// Start launches the Cloud Hypervisor VM in the background.
func (a *CloudHypervisorAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.binary, "--api-socket", "/tmp/"+id+".sock", "&")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloud-hypervisor start %s: %w: %s", id, err, string(out))
	}
	return nil
}

// Stop terminates the Cloud Hypervisor VM via pkill.
func (a *CloudHypervisorAdapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", "cloud-hypervisor.*"+id)
	return cmd.Run()
}

// Delete stops the VM and releases its state.
func (a *CloudHypervisorAdapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// GetStartupTime returns the typical Cloud Hypervisor boot latency.
func (a *CloudHypervisorAdapter) GetStartupTime() time.Duration { return 180 * time.Millisecond }

// Probe verifies the cloud-hypervisor binary is on $PATH and we are on
// a supported platform (Linux today).
func (a *CloudHypervisorAdapter) Probe(_ context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("cloudhv: platform %q not supported (linux-only)", runtime.GOOS)
	}
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("cloudhv: binary %q not found: %w", a.binary, err)
	}
	return nil
}
