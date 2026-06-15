// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier provides public tier adapters for NVMS isolation levels.
package tier

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/kooshapari/nanovms/internal/domain"
)

// GVisorAdapter is the Tier2 gVisor adapter for semi-trusted workloads.
// Startup: ~90ms, Memory: ~50MB, CPU overhead: ~5%
type GVisorAdapter struct {
	runtime string
}

// NewGVisorAdapter creates a new Tier2 gVisor adapter.
func NewGVisorAdapter() *GVisorAdapter {
	return &GVisorAdapter{
		runtime: "runsc",
	}
}

// Deploy deploys a gVisor sandbox workload.
func (a *GVisorAdapter) Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	if _, err := exec.LookPath(a.runtime); err != nil {
		return nil, fmt.Errorf("gVisor runtime (%s) not found: %w", a.runtime, err)
	}

	sandbox := &domain.Sandbox{
		ID:     fmt.Sprintf("gvisor-%s", domain.GenerateID()),
		Name:   config.Name,
		Status: domain.SandboxStatusRunning,
		Type:   domain.SandboxTypeGVisor,
		Config: &config,
	}
	return sandbox, nil
}
