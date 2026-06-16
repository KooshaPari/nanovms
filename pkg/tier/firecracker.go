// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier provides public tier adapters for NVMS isolation levels.
package tier

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// FirecrackerAdapter is the Tier3 Firecracker microVM adapter for untrusted workloads.
// Startup: ~125ms, Memory: ~128MB, CPU overhead: ~10%
type FirecrackerAdapter struct {
	path      string
	apiSocket string
}

// NewFirecrackerAdapter creates a new Tier3 Firecracker adapter.
func NewFirecrackerAdapter() *FirecrackerAdapter {
	return &FirecrackerAdapter{
		apiSocket: "/tmp/firecracker-api.sock",
	}
}

// Deploy deploys a Firecracker microVM workload.
func (a *FirecrackerAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	path, err := exec.LookPath("firecracker")
	if err != nil {
		return nil, fmt.Errorf("firecracker binary not found: %w", err)
	}
	a.path = path

	id := fmt.Sprintf("fc-%s", domain.GenerateID())
	sandbox := &domain.Sandbox{
		ID:       id,
		Name:     config.Name,
		Status:   domain.SandboxStatusRunning,
		Type:     domain.SandboxTypeVM,
		VMFlavor: domain.VMFlavorMicroVM,
		Config:   &config,
		CreatedAt: time.Now(),
	}
	return sandbox, nil
}

// Start starts the Firecracker microVM.
func (a *FirecrackerAdapter) Start(ctx context.Context, id string) error {
	if a.path == "" {
		return fmt.Errorf("firecracker path not set")
	}
	// Firecracker starts via API socket after the binary is launched
	cmd := exec.CommandContext(ctx, a.path, "--api-sock", a.apiSocket, "--id", id)
	return cmd.Start()
}

// Stop stops the Firecracker microVM.
func (a *FirecrackerAdapter) Stop(ctx context.Context, id string) error {
	// Send shutdown via API socket or kill the process
	cmd := exec.CommandContext(ctx, "pkill", "-f", "firecracker.*"+id)
	return cmd.Run()
}

// Delete deletes the Firecracker microVM.
func (a *FirecrackerAdapter) Delete(ctx context.Context, id string) error {
	// Clean up socket and VM state
	if err := a.Stop(ctx, id); err != nil {
		return fmt.Errorf("failed to stop vm %s: %w", id, err)
	}
	return nil
}

// GetStartupTime returns the typical startup time for a Firecracker microVM.
func (a *FirecrackerAdapter) GetStartupTime() time.Duration {
	return 125 * time.Millisecond
}
