// SPDX-License-Identifier: MIT OR Apache-2.0
// Package orchestrate provides the orchestration engine that dispatches workloads to tiers.
package orchestrate

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/config"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// Engine is the orchestration engine that routes workloads to the appropriate tier.
type Engine struct {
	tier1 tier1Runtime
	tier2 tier2Runtime
	tier3 tier3Runtime
}

// tier1Runtime is the interface for Tier1 (WASM) workloads.
type tier1Runtime interface {
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)
}

// tier2Runtime is the interface for Tier2 (gVisor) workloads.
type tier2Runtime interface {
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)
}

// tier3Runtime is the interface for Tier3 (Firecracker) workloads.
type tier3Runtime interface {
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)
}

// NewEngine creates a new orchestration engine with default tier adapters.
func NewEngine() *Engine {
	return &Engine{
		tier1: tier.NewWASMAdapter(),
		tier2: tier.NewGVisorAdapter(),
		tier3: tier.NewFirecrackerAdapter(),
	}
}

// NewEngineWithAdapters creates a new orchestration engine with custom tier adapters.
func NewEngineWithAdapters(t1 tier1Runtime, t2 tier2Runtime, t3 tier3Runtime) *Engine {
	return &Engine{
		tier1: t1,
		tier2: t2,
		tier3: t3,
	}
}

// DeployFromConfig deploys a workload using an NVMS configuration file.
func (e *Engine) DeployFromConfig(ctx context.Context, cfg *config.NVMSConfig) (*domain.Sandbox, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	domainCfg := cfg.ToDomainConfig()
	return e.Deploy(ctx, cfg.Tier, domainCfg)
}

// Deploy deploys a workload to the specified tier.
func (e *Engine) Deploy(ctx context.Context, tierLevel int, config domain.SandboxConfig) (*domain.Sandbox, error) {
	start := time.Now()

	var sandbox *domain.Sandbox
	var err error

	switch tierLevel {
	case 1:
		sandbox, err = e.tier1.Deploy(ctx, config)
	case 2:
		sandbox, err = e.tier2.Deploy(ctx, config)
	case 3:
		sandbox, err = e.tier3.Deploy(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported tier level: %d", tierLevel)
	}

	if err != nil {
		return nil, fmt.Errorf("tier %d deployment failed: %w", tierLevel, err)
	}

	// Record deployment metadata
	sandbox.Environment = mergeLabels(sandbox.Environment, map[string]string{
		"nvms.tier":       fmt.Sprintf("%d", tierLevel),
		"nvms.deployedAt": start.Format(time.RFC3339),
	})

	return sandbox, nil
}

// Stop stops a running sandbox by tier.
func (e *Engine) Stop(ctx context.Context, tierLevel int, id string) error {
	switch tierLevel {
	case 1:
		return fmt.Errorf("tier1 stop not yet implemented for id=%s", id)
	case 2:
		return fmt.Errorf("tier2 stop not yet implemented for id=%s", id)
	case 3:
		if fc, ok := e.tier3.(*tier.FirecrackerAdapter); ok {
			return fc.Stop(ctx, id)
		}
		return fmt.Errorf("tier3 stop not available for id=%s", id)
	default:
		return fmt.Errorf("unsupported tier level: %d", tierLevel)
	}
}

// Delete deletes a sandbox by tier.
func (e *Engine) Delete(ctx context.Context, tierLevel int, id string) error {
	switch tierLevel {
	case 1:
		return fmt.Errorf("tier1 delete not yet implemented for id=%s", id)
	case 2:
		return fmt.Errorf("tier2 delete not yet implemented for id=%s", id)
	case 3:
		if fc, ok := e.tier3.(*tier.FirecrackerAdapter); ok {
			return fc.Delete(ctx, id)
		}
		return fmt.Errorf("tier3 delete not available for id=%s", id)
	default:
		return fmt.Errorf("unsupported tier level: %d", tierLevel)
	}
}

// mergeLabels merges two label maps, with the second taking precedence.
func mergeLabels(base, override map[string]string) map[string]string {
	if base == nil {
		base = make(map[string]string)
	}
	for k, v := range override {
		base[k] = v
	}
	return base
}
