// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — jail_other.go is the non-freebsd stub for registerJail.
// The matching jail.go (//go:build freebsd) provides the real adapter
// and registration; this file guarantees the function symbol exists on
// every platform so defaults.go can call registerJail unconditionally.
//go:build !freebsd

package tier

// registerJail is a no-op on non-freebsd platforms. The jail adapter
// itself is gated to freebsd (jail.go). Calling DefaultRegistry on a
// Linux or macOS host returns a registry that does not advertise the
// "jail" tier; an explicit NVMS_TIER=jail request returns an "unknown
// tier" error from envExactTier.
func registerJail(_ *Registry) {}
