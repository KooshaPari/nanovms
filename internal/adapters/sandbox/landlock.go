// Package sandbox / landlock.go — Linux Landlock adapter.
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"github.com/kooshapari/nanovms/internal/domain"
)

type landlockAdapter struct {
	noNewPrivs bool
}
func (a *landlockAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeLandlock,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for landlockAdapter.
//
// On Linux kernels with landlock (>= 5.13), this installs a minimal
// ruleset and restricts the calling thread to it (see landlock_linux.go
// for the actual syscall sequence). After landlock_restrict_self the
// thread and all descendants are confined to the allowed paths for
// the lifetime of the process — there is no way to escape the ruleset
// from user-space.
//
// On kernels without landlock, or on non-Linux platforms, Start returns
// an error and the caller is expected to refuse to run the workload
// (see T-NV.2.7).
func (a *landlockAdapter) Start(ctx context.Context, id string) error {
	if !kernelSupportsLandlockWrapper() {
		return fmt.Errorf("landlock: kernel does not support landlock (need Linux >= 5.13); refusing to start sandbox %q", id)
	}

	readOnlyPaths := []string{
		"/usr", "/usr/lib", "/usr/lib64", "/usr/share",
		"/lib", "/lib64",
		"/etc", "/etc/ssl", "/etc/resolv.conf",
		"/bin", "/sbin",
		"/var/lib/dpkg", "/var/lib/rpm",
	}
	readWritePaths := []string{
		"/tmp",
	}

	rulesetFd, err := buildLandlockRulesetStub(readOnlyPaths, readWritePaths)
	if err != nil {
		return fmt.Errorf("landlock: build ruleset: %w", err)
	}
	if err := landlockRestrictSelfStub(rulesetFd); err != nil {
		return fmt.Errorf("landlock: restrict_self: %w", err)
	}
	return nil
}

// Stop implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Delete(ctx context.Context, id string) error {
	return nil // Landlock rules are cleaned up with the process
}

// Create implements ports.SandboxPort for seccompAdapter.
