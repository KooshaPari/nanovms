// SPDX-License-Identifier: MIT OR Apache-2.0
// Package orchestrate provides the orchestration engine that dispatches workloads to tiers.
package orchestrate

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters"
	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/config"
	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// Engine is the orchestration engine that routes workloads to the appropriate tier.
type Engine struct {
	tier1 tier1Runtime
	tier2 tier2Runtime
	tier3 tier3Runtime

	backendDispatchers map[nvmsruntime.BackendID]BackendDispatcher
	reservations       *gpu.ReservationStore
	reservationTTL     time.Duration
}

// BackendDispatcher is a backend-specific lifecycle implementation. Backend
// metadata is never translated into a generic tier dispatcher.
type BackendDispatcher interface {
	Deploy(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)
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
	tier3 := tier.NewFirecrackerAdapter()
	dispatchers := map[nvmsruntime.BackendID]BackendDispatcher{
		nvmsruntime.BackendNanoVMS: tier3,
	}
	for backend, provider := range map[nvmsruntime.BackendID]string{
		nvmsruntime.BackendPodman:          "podman",
		nvmsruntime.BackendAppleContainers: "apple-containers",
		nvmsruntime.BackendWSLContainers:   "wsl-containers",
	} {
		if port, err := adapters.NewProvider(provider, 2); err == nil {
			dispatchers[backend] = newSandboxPortDispatcher(port)
		}
	}
	return &Engine{
		tier1:              tier.NewWASMAdapter(),
		tier2:              tier.NewGVisorAdapter(),
		tier3:              tier3,
		backendDispatchers: dispatchers,
		reservationTTL:     15 * time.Minute,
	}
}

// NewEngineWithAdapters creates a new orchestration engine with custom tier adapters.
func NewEngineWithAdapters(t1 tier1Runtime, t2 tier2Runtime, t3 tier3Runtime) *Engine {
	return &Engine{
		tier1: t1,
		tier2: t2,
		tier3: t3,
		backendDispatchers: map[nvmsruntime.BackendID]BackendDispatcher{
			nvmsruntime.BackendNanoVMS: t3,
		},
		reservationTTL: 15 * time.Minute,
	}
}

// RegisterBackendDispatcher installs an implementation for one exact backend.
func (e *Engine) RegisterBackendDispatcher(backend nvmsruntime.BackendID, dispatcher BackendDispatcher) error {
	if e == nil || dispatcher == nil {
		return fmt.Errorf("engine and backend dispatcher are required")
	}
	metadata, err := nvmsruntime.NewBackendRegistry().Resolve(backend)
	if err != nil {
		return err
	}
	if !metadata.Lifecycle {
		return fmt.Errorf("backend %q does not advertise lifecycle support", backend)
	}
	if e.backendDispatchers == nil {
		e.backendDispatchers = make(map[nvmsruntime.BackendID]BackendDispatcher)
	}
	e.backendDispatchers[backend] = dispatcher
	return nil
}

// ConfigureGPUReservations enables resource-aware deployment reservations.
func (e *Engine) ConfigureGPUReservations(store *gpu.ReservationStore, ttl time.Duration) error {
	if e == nil || store == nil || ttl <= 0 {
		return fmt.Errorf("engine, reservation store, and positive expiry are required")
	}
	e.reservations = store
	e.reservationTTL = ttl
	return nil
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
		// Tiers 4-30: route through the generic SandboxPort adapter layer.
		port, portErr := adapters.NewSandboxPort(tierLevel)
		if portErr != nil {
			return nil, fmt.Errorf("tier %d: %w", tierLevel, portErr)
		}
		sandbox, err = port.Create(ctx, config)
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
		port, portErr := adapters.NewSandboxPort(tierLevel)
		if portErr != nil {
			return fmt.Errorf("tier %d: %w", tierLevel, portErr)
		}
		return port.Stop(ctx, id, false)
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
		port, portErr := adapters.NewSandboxPort(tierLevel)
		if portErr != nil {
			return fmt.Errorf("tier %d: %w", tierLevel, portErr)
		}
		return port.Delete(ctx, id)
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
