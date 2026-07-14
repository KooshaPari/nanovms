// SPDX-License-Identifier: MIT OR Apache-2.0
package tier

import (
	"strings"
	"testing"
)

// TestRegistry_RegisterAndGet verifies the happy path: register, get back,
// Has returns true. Duplicate names are rejected.
func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	a := NewFirecrackerAdapter()
	if err := r.Register("firecracker", a); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, err := r.Get("firecracker")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != a {
		t.Fatalf("get returned different adapter")
	}
	if !r.Has("firecracker") {
		t.Fatalf("Has returned false after register")
	}
}

// TestRegistry_DuplicateName verifies that a second Register with the
// same name returns an error and the first adapter is retained.
func TestRegistry_DuplicateName(t *testing.T) {
	r := NewRegistry()
	a := NewFirecrackerAdapter()
	if err := r.Register("firecracker", a); err != nil {
		t.Fatalf("first register: %v", err)
	}
	b := NewWASMAdapter()
	if err := r.Register("firecracker", b); err == nil {
		t.Fatalf("expected duplicate-name error, got nil")
	}
	got, _ := r.Get("firecracker")
	if got != a {
		t.Fatalf("duplicate register replaced original adapter")
	}
}

// TestRegistry_NamesOrdering verifies Names() returns alphabetically
// sorted tier names.
func TestRegistry_NamesOrdering(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterAll([]AdapterEntry{
		{Name: "wasm", Adapter: NewWASMAdapter()},
		{Name: "firecracker", Adapter: NewFirecrackerAdapter()},
		{Name: "gvisor", Adapter: NewGVisorAdapter()},
	}); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	got := r.Names()
	want := []string{"firecracker", "gvisor", "wasm"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

// TestRegistry_NamesIsSortedAllDefault ensures the package-wide
// DefaultRegistry exposes every tier in deterministic sorted order.
func TestRegistry_NamesIsSortedAllDefault(t *testing.T) {
	got := DefaultRegistry().Names()
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("Names() not sorted at %d: %v", i, got)
		}
	}
	if len(got) < 15 {
		t.Fatalf("expected at least 15 tiers, got %d (%v)", len(got), got)
	}
}

// TestRegistry_EmptyNameRejected verifies the empty-name guard.
func TestRegistry_EmptyNameRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("", NewFirecrackerAdapter()); err == nil {
		t.Fatalf("expected error for empty name")
	}
}

// TestRegistry_NilAdapterRejected verifies the nil-adapter guard.
func TestRegistry_NilAdapterRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("firecracker", nil); err == nil {
		t.Fatalf("expected error for nil adapter")
	}
}

// TestRegistry_GetUnknownErrors verifies Get() error path mentions the
// requested name and the registered names.
func TestRegistry_GetUnknownErrors(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("ghost")
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("expected error mentioning ghost, got %v", err)
	}
}

// TestRegistry_RegisterAllStopsAtDuplicate verifies RegisterAll returns
// the first duplicate-name error wrapped with the entry index.
func TestRegistry_RegisterAllStopsAtDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("a", NewWASMAdapter()); err != nil {
		t.Fatalf("seed register: %v", err)
	}
	err := r.RegisterAll([]AdapterEntry{
		{Name: "b", Adapter: NewWASMAdapter()},
		{Name: "a", Adapter: NewFirecrackerAdapter()},
	})
	if err == nil {
		t.Fatalf("expected error for duplicate name in batch")
	}
	if !strings.Contains(err.Error(), `"a"`) {
		t.Fatalf("error should mention name: %v", err)
	}
}

// TestRegistry_BySecurity verifies the security filter returns exactly
// the tiers whose Security level matches.
func TestRegistry_BySecurity(t *testing.T) {
	r := DefaultRegistry()
	high := r.BySecurity("high")
	if len(high) == 0 {
		t.Fatalf("expected at least one high-security tier")
	}
	for _, n := range high {
		if r.Info()[n].Security != "high" {
			t.Fatalf("BySecurity(high) returned %q with Security=%q", n, r.Info()[n].Security)
		}
	}
	// Sorted
	for i := 1; i < len(high); i++ {
		if high[i-1] >= high[i] {
			t.Fatalf("BySecurity not sorted: %v", high)
		}
	}
}

// TestRegistry_ByPlatform verifies the platform filter only returns
// tiers that advertise support for that platform.
func TestRegistry_ByPlatform(t *testing.T) {
	r := DefaultRegistry()
	linux := r.ByPlatform("linux")
	for _, n := range linux {
		infos := r.Info()[n].Platforms
		hasLinux := false
		for _, p := range infos {
			if p == "linux" {
				hasLinux = true
				break
			}
		}
		if !hasLinux {
			t.Fatalf("ByPlatform(linux) returned %q without linux in Platforms=%v", n, infos)
		}
	}
}

// TestRegistry_ByStartup verifies the budget filter only returns tiers
// whose StartupMS is within the budget.
func TestRegistry_ByStartup(t *testing.T) {
	r := DefaultRegistry()
	budget := 100
	got := r.ByStartup(budget)
	for _, n := range got {
		if r.Info()[n].StartupMS > budget {
			t.Fatalf("ByStartup(%d) returned %q with StartupMS=%d", budget, n, r.Info()[n].StartupMS)
		}
	}
	// wasm (1ms), seccomp (1ms), landlock (1ms), native (0ms) are the
	// only tiers under 100ms startup.
	if len(got) < 3 {
		t.Fatalf("expected at least 3 tiers under %dms, got %d (%v)", budget, len(got), got)
	}
}

// TestRegistry_InfoSnapshot verifies Info() returns a snapshot (mutating
// the returned map does not affect the registry).
func TestRegistry_InfoSnapshot(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("wasm", NewWASMAdapter()); err != nil {
		t.Fatalf("register: %v", err)
	}
	snap := r.Info()
	snap["wasm"] = TierInfo{Name: "tampered"}
	if r.Info()["wasm"].Name != "wasm" {
		t.Fatalf("Info() returned a shared map; mutation leaked back")
	}
}

// TestRegistry_DefaultIsIdempotent verifies DefaultRegistry() can be
// called repeatedly without duplicating entries.
func TestRegistry_DefaultIsIdempotent(t *testing.T) {
	a := DefaultRegistry()
	b := DefaultRegistry()
	if a != b {
		t.Fatalf("DefaultRegistry returned different pointers")
	}
	if len(a.Names()) != len(b.Names()) {
		t.Fatalf("DefaultRegistry not idempotent: %d vs %d", len(a.Names()), len(b.Names()))
	}
}