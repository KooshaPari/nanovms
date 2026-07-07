// Package sandbox / gvisor.go — gVisor (runsc) adapter.
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"github.com/kooshapari/nanovms/internal/domain"
)

type gvisorAdapter struct {
	runtime   string
	overlayFS bool
}
func (a *gvisorAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()

	cmd := exec.CommandContext(ctx, a.runtime,
		"run",
		"--id", id,
	)
	if a.overlayFS {
		cmd.Args = append(cmd.Args, "--overlay", runscPath, "/")
	} else {
		cmd.Args = append(cmd.Args, "--read-only", runscPath, "/")
	}

	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeGVisor,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.runtime, "kill", "-SIGCONT", id)
	return cmd.Run()
}

// Stop implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Stop(ctx context.Context, id string, force bool) error {
	signal := "SIGTERM"
	if force {
		signal = "SIGKILL"
	}
	cmd := exec.CommandContext(ctx, a.runtime, "kill", "-"+signal, id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	return nil
}

// Delete implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.runtime, "delete", id)
	return cmd.Run()
}

// Create implements ports.SandboxPort for landlockAdapter.
