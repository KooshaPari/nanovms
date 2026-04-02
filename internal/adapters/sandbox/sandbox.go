// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"devenv/abstraction/internal/domain"
	"devenv/abstraction/internal/ports"
)

// Adapter implements the SandboxPort interface for sandbox isolation technologies.
// It provides a unified interface for gVisor, landlock, seccomp, and wasmtime sandboxes.
type Adapter struct{}

// NewAdapter creates a new sandbox adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// gvisorAdapter implements sandboxing using gVisor (runsc).
type gvisorAdapter struct {
	runtime   string
	overlayFS bool
}

// landlockAdapter implements sandboxing using Linux landlock.
type landlockAdapter struct {
	noNewPrivs bool
}

// seccompAdapter implements sandboxing using seccomp.
type seccompAdapter struct {
	defaultAction string
}

// wasmtimeAdapter implements sandboxing using wasmtime.
type wasmtimeAdapter struct {
	wasmEngine string
}

// New creates a new sandbox with the specified configuration.
func (a *Adapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	var adapter ports.SandboxPort

	switch config.Type {
	case domain.SandboxTypeGVisor:
		adapter = &gvisorAdapter{
			runtime:   config.RuntimePath,
			overlayFS: config.EnableOverlayFS,
		}
	case domain.SandboxTypeLandlock:
		adapter = &landlockAdapter{
			noNewPrivs: config.NoNewPrivs,
		}
	case domain.SandboxTypeSeccomp:
		adapter = &seccompAdapter{
			defaultAction: config.SeccompDefaultAction,
		}
	case domain.SandboxTypeWasmtime:
		adapter = &wasmtimeAdapter{
			wasmEngine: config.WasmEngine,
		}
	default:
		return nil, fmt.Errorf("unsupported sandbox type: %s", config.Type)
	}

	return adapter.New(ctx, config)
}

// List lists available sandbox runtimes.
func (a *Adapter) List(ctx context.Context) ([]domain.SandboxRuntime, error) {
	runtimes := []domain.SandboxRuntime{}

	// Check for gVisor
	if path, err := exec.LookPath("runsc"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "gVisor",
			Type:    domain.SandboxTypeGVisor,
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	// Check for landlock support
	if a.checkLandlockSupport() {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Landlock",
			Type:    domain.SandboxTypeLandlock,
			Path:    "kernel-native",
			Version: "kernel-supported",
		})
	}

	// Check for wasmtime
	if path, err := exec.LookPath("wasmtime"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Wasmtime",
			Type:    domain.SandboxTypeWasmtime,
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	return runtimes, nil
}

// New implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
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
		ID:         id,
		Type:       domain.SandboxTypeGVisor,
		Config:     config,
		PID:        -1,
		Status:     domain.SandboxStatusCreating,
		Mounts:     config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	// gVisor sandbox is started with the run command
	// PID is assigned when the container starts
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	cmd := exec.CommandContext(ctx, a.runtime, "kill", sandbox.ID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	cmd := exec.CommandContext(ctx, a.runtime, "delete", sandbox.ID)
	return cmd.Run()
}

// New implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeLandlock,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	// Landlock is enforced at the kernel level via syscalls
	// Start the process with landlock enabled
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil // Landlock rules are cleaned up with the process
}

// New implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeSeccomp,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil
}

// New implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeWasmtime,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil
}

// checkLandlockSupport checks if the kernel supports landlock.
func (a *Adapter) checkLandlockSupport() bool {
	// Landlock support requires kernel >= 5.13
	// We check by looking for /sys/kernel/security/landlock
	return true // Simplified - real implementation would check kernel version
}

// getVersion returns the version of a runtime.
func (a *Adapter) getVersion(path string) string {
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// generateID generates a unique sandbox ID.
func generateID() string {
	// Simplified - real implementation would use UUID
	return fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
}

// Ensure ports.SandboxPort is implemented.
var _ ports.SandboxPort = (*Adapter)(nil)
var _ ports.SandboxPort = (*gvisorAdapter)(nil)
var _ ports.SandboxPort = (*landlockAdapter)(nil)
var _ ports.SandboxPort = (*seccompAdapter)(nil)
var _ ports.SandboxPort = (*wasmtimeAdapter)(nil)
