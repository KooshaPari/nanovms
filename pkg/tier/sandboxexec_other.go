// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — sandboxexec_other.go is the non-darwin stub for
// registerSandboxExec. The matching sandboxexec.go (//go:build darwin)
// provides the real adapter and registration; this file guarantees the
// function symbol exists on every platform so defaults.go can call
// registerSandboxExec unconditionally.
//go:build !darwin

package tier

// registerSandboxExec is a no-op on non-darwin platforms. The
// sandbox-exec adapter itself is gated to darwin (sandboxexec.go).
func registerSandboxExec(_ *Registry) {}
