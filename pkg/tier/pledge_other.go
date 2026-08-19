// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — pledge_other.go is the non-openbsd stub for
// registerPledge. The matching pledge.go (//go:build openbsd) provides
// the real adapter and registration; this file guarantees the function
// symbol exists on every platform so defaults.go can call registerPledge
// unconditionally.
//go:build !openbsd

package tier

// registerPledge is a no-op on non-openbsd platforms. The pledge
// adapter itself is gated to openbsd (pledge.go).
func registerPledge(_ *Registry) {}
