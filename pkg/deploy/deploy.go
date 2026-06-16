// SPDX-License-Identifier: MIT OR Apache-2.0
// Package deploy provides the deploy orchestrator for NVMS tiered deployments.
package deploy

import (
	"context"
	"fmt"
	"os"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// Config represents a deployment configuration.
type Config struct {
	Name   string `yaml:"name" json:"name"`
	Image  string `yaml:"image" json:"image"`
	Tier   int    `yaml:"tier" json:"tier"`
	CPU    int    `yaml:"cpu" json:"cpu"`
	Memory int    `yaml:"memory" json:"memory"`
}

// Deploy orchestrates a deployment to the specified tier.
func Deploy(ctx context.Context, tierLevel int, configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	switch tierLevel {
	case 1:
		return deployTier1(ctx, configPath)
	case 2:
		return deployTier2(ctx, configPath)
	case 3:
		return deployTier3(ctx, configPath)
	default:
		return fmt.Errorf("unsupported tier: %d", tierLevel)
	}
}

func deployTier1(ctx context.Context, configPath string) error {
	adapter := tier.NewWASMAdapter()
	config := domain.SandboxConfig{
		Name: "wasm-workload",
	}
	_, err := adapter.Deploy(ctx, config)
	return err
}

func deployTier2(ctx context.Context, configPath string) error {
	adapter := tier.NewGVisorAdapter()
	config := domain.SandboxConfig{
		Name: "gvisor-workload",
	}
	_, err := adapter.Deploy(ctx, config)
	return err
}

func deployTier3(_ context.Context, _ string) error {
	return fmt.Errorf("tier3 (Firecracker) not yet implemented")
}
