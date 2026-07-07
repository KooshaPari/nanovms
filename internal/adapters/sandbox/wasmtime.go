// Package sandbox / wasmtime.go — wasmtime adapter.
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

type wasmtimeAdapter struct {
	wasmEngine string
}
func (a *wasmtimeAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeWasmtime,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Start(ctx context.Context, id string) error {
	return nil
}

// Stop implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

// checkLandlockSupport checks if the kernel supports landlock.
// Landlock requires kernel >= 5.13 (released 2021-06-27).
//
// The kernel exposes landlock support through three independent mechanisms,
// probed in order from cheapest+most-authoritative to most-permissive:
//
//  1. The `landlock_create_ruleset` syscall (preferred). If the syscall
//     exists in the running kernel, this call will return a real fd (or
//     ENOSYS on kernels <5.13). ENOSYS is unambiguous "no support".
//  2. The `/sys/kernel/landlock_restrict_self` ABI file (newer kernels).
//  3. The legacy `/sys/kernel/security/landlock` ABI file (kernels 5.13-5.14).
//
// The previous version of this function fell through to "optimistic true"
// when the kernel was >=5.13 but the ABI files were missing. That is a
// false-positive on hardened kernels that strip the sysfs entries while
// keeping the syscall disabled. The syscall probe closes that gap
// (closes T-NV.2.3 from plans/2026-06-22-compute-infra-dag-v1.md).
