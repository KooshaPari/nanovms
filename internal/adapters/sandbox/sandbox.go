// Package sandbox — adapter decomposition root.
// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os/exec"
	"github.com/kooshapari/nanovms/internal/domain"
)


// cryptoRandReader is the global random reader used for ID generation.
var cryptoRandReader io.Reader = rand.Reader

// runscPath is the path to the runsc binary (gVisor runtime).
var runscPath = "/usr/local/bin/runsc"

// Adapter implements the SandboxPort interface for sandbox isolation technologies.
// It provides a unified interface for gVisor, landlock, seccomp, and wasmtime sandboxes.

type Adapter struct {
	sandboxes map[string]*domain.Sandbox
}

// NewAdapter creates a new sandbox adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		sandboxes: make(map[string]*domain.Sandbox),
	}
}


// List implements ports.SandboxPort for Adapter.
func (a *Adapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	result := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, sb := range a.sandboxes {
		result = append(result, sb)
	}
	return result, nil
}

// Get implements ports.SandboxPort for Adapter.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return sb, nil
}

// Logs implements ports.SandboxPort for Adapter.
// Returns logs by delegating to the native sandbox adapter if available,
// or by querying the runtime's log mechanism.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return logsForSandbox(ctx, sb)
}

// Exec implements ports.SandboxPort for Adapter.
// Executes a command in the specified sandbox using the native adapter.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return execInSandbox(ctx, sb, cmd)
}

// Metrics implements ports.SandboxPort for Adapter.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return metricsForSandbox(ctx, sb)
}

// List implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for gvisorAdapter.
// Note: gvisorAdapter does not maintain local sandbox storage.
// The caller must track sandbox IDs and re-create the adapter as needed.
func (a *gvisorAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Verify the sandbox still exists by querying runc
	cmd := exec.CommandContext(ctx, a.runtime, "ps", id)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	// Construct a minimal sandbox reference (PIDs tracked externally)
	return &domain.Sandbox{
		ID:     id,
		Status: domain.SandboxStatusRunning,
		Type:   domain.SandboxTypeGVisor,
	}, nil
}

// Logs implements ports.SandboxPort for gvisorAdapter.
// Retrieves logs via runsc log command or journald.
func (a *gvisorAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{runscPath, "logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, id)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "GvisorRuntime="+a.runtime)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runsc logs: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for gvisorAdapter.
// Executes a command in a running gVisor sandbox via runsc exec.
func (a *gvisorAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	args := []string{runscPath, "exec"}
	args = append(args, id)
	args = append(args, "--")
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "GvisorRuntime="+a.runtime)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runsc exec: %w", err)
	}
	return r, nil
}

// Metrics implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	metrics := &domain.SandboxMetrics{SandboxID: id}
	// Query runc for process stats
	cmd := exec.CommandContext(ctx, a.runtime, "ps", id)
	out, err := cmd.Output()
	if err != nil {
		return metrics, nil // Return empty metrics if query fails
	}
	// Parse output (simplified - production would parse full ps output)
	if len(out) > 0 {
		metrics.CPUUsage = 0 // Would be parsed from runc stats
	}
	return metrics, nil
}

// List implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for landlockAdapter.
// Landlock is enforced at the kernel level; sandboxes are tracked by their PIDs.
// Use the PID stored when the sandboxed process was started.
func (a *landlockAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Landlock sandboxes are tracked via PID files or external process management.
	// Return a placeholder; real implementation would read PID from a tracking file.
	// Field `noNewPrivs` is reserved for the future landlock_ruleset-based
	// restriction enforcement (kernel >= 5.13); left referenced here so the
	// field stays part of the public surface and so a future PR can wire it
	// up without a struct-shape migration.
	_ = a.noNewPrivs // suppress unused warning
	return nil, fmt.Errorf("landlock sandbox must be tracked externally by PID for id=%s", id)
}

// Logs implements ports.SandboxPort for landlockAdapter.
// Landlock sandboxes write logs via the container runtime or journald.
func (a *landlockAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"journalctl", "-t", "landlock-" + id}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for landlockAdapter.
// Executes in the landlock sandbox by finding the process and using nsenter.
func (a *landlockAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	// Landlock sandboxes must be tracked externally; exec via nsenter.
	// For now, return not implemented with guidance.
	_ = a.noNewPrivs
	return nil, fmt.Errorf("landlock sandbox exec requires external PID tracking; use nsenter with known PID for id=%s", id)
}

// Metrics implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.noNewPrivs
	return &domain.SandboxMetrics{SandboxID: id}, nil
}

// List implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for seccompAdapter.
// Seccomp sandboxes are enforced at the process level; tracked externally.
func (a *seccompAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Seccomp is applied via prctl; sandbox state is managed by the parent process.
	return nil, fmt.Errorf("seccomp sandbox tracked externally; use container runtime for id=%s", id)
}

// Logs implements ports.SandboxPort for seccompAdapter.
// Seccomp sandboxes log via the controlling runtime (runc, containerd, etc.).
func (a *seccompAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"journalctl", "-t", "seccomp-" + id}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	_ = a.defaultAction
	return nil, fmt.Errorf("seccomp sandbox exec requires container runtime; use runc exec for id=%s", id)
}

// Metrics implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.defaultAction
	return &domain.SandboxMetrics{SandboxID: id}, nil
}

// List implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for wasmtimeAdapter.
// wasmtimeAdapter does not maintain sandbox state; modules are stateless WASM instances.
func (a *wasmtimeAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// WASM instances are stateless; module state is in the running process.
	// The caller should track WASM module IDs separately.
	return nil, fmt.Errorf("WASM sandbox state tracked by caller; use wasmtime instance API for id=%s", id)
}

// Logs implements ports.SandboxPort for wasmtimeAdapter.
// WASM modules do not produce traditional logs; stderr is captured via wasmtime.
func (a *wasmtimeAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	// WASM stderr is redirected by the calling runtime.
	// Use the wasmtime --env flag to capture stderr, or check the calling process.
	_ = a.wasmEngine
	return nil, fmt.Errorf("WASM logs must be captured by the calling runtime; use wasmtime with --dir for id=%s", id)
}

// Exec implements ports.SandboxPort for wasmtimeAdapter.
// Executes a WASM module via wasmtime with the given command-line arguments.
func (a *wasmtimeAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	args := []string{"wasmtime"}
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start wasmtime: %w", err)
	}
	return r, nil
}

// Metrics implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.wasmEngine
	return &domain.SandboxMetrics{SandboxID: id}, nil
}

// List implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	result := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, s := range a.sandboxes {
		result = append(result, s)
	}
	return result, nil
}

// Get implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return sandbox, nil
}

// Logs implements ports.SandboxPort for nativeSandboxAdapter.
// Retrieves logs from a running native sandbox via journald.
func (a *nativeSandboxAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return logsForSandbox(ctx, sb)
}

// Exec implements ports.SandboxPort for nativeSandboxAdapter.
// Executes a command in the native sandbox's namespace via nsenter.
func (a *nativeSandboxAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return execInSandbox(ctx, sb, cmdArgs)
}

// Metrics implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return &domain.SandboxMetrics{
		SandboxID:   sandbox.ID,
		CPUUsage:    0,
		MemoryUsage: 0,
	}, nil
}
