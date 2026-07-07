// Package sandbox / adapter.go — Adapter facade methods + helpers.
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"github.com/kooshapari/nanovms/internal/domain"
)

func (a *Adapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	now := time.Now()
	sandbox := &domain.Sandbox{
		ID:        id,
		Name:      config.Name,
		Status:    domain.SandboxStatusPending,
		VMFlavor:  config.VMType,
		CreatedAt: now,
	}
	a.sandboxes[id] = sandbox
	return sandbox, nil
}

// Start implements ports.SandboxPort for Adapter.
func (a *Adapter) Start(ctx context.Context, id string) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	now := time.Now()
	sandbox.Status = domain.SandboxStatusRunning
	sandbox.StartedAt = &now
	return nil
}

// Stop implements ports.SandboxPort for Adapter.
func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for Adapter.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	delete(a.sandboxes, id)
	return nil
}

// ListRuntimes lists available sandbox runtimes.
func (a *Adapter) ListRuntimes(ctx context.Context) ([]domain.SandboxRuntime, error) {
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

// Create implements ports.SandboxPort for gvisorAdapter.


func (a *Adapter) checkLandlockSupport() bool {
	// 1. Authoritative: ask the kernel directly via the probe wrapper.
	//    The wrapper does the syscall + ABI fallback chain.
	if kernelSupportsLandlockWrapper() {
		return true
	}

	// 2. Last resort: ABI file checks (some kernels have the ABI files
	//    but not the syscall — extremely rare but documented).
	if _, err := os.Stat("/sys/kernel/landlock_restrict_self"); err == nil {
		return true
	}
	if _, err := os.Stat("/sys/kernel/security/landlock"); err == nil {
		return true
	}

	return false
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

// logsForSandbox retrieves logs for a running sandbox by PID.
