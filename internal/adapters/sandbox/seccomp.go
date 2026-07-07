// Package sandbox / seccomp.go — seccomp adapter.
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"github.com/kooshapari/nanovms/internal/domain"
)

type seccompAdapter struct {
	defaultAction string
}
func (a *seccompAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeSeccomp,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Start(ctx context.Context, id string) error {
	return nil
}

// Stop implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

// Create implements ports.SandboxPort for wasmtimeAdapter.
