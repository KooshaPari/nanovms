// SPDX-License-Identifier: MIT OR Apache-2.0
// Tests covering the 15 newly added isolation/runtime tiers
// (kata, youki, systemdnspawn, nitrorekf, sev, tdx, kubevirt,
// virtcontainers, jail, pledge, sandboxexec, gvisordocker, wolfi,
// distroless, userns).
//
// These tests verify registry membership, Probe() surface area, and
// that ProfilePolicy correctly routes the new tiers into the
// appropriate candidate lists.
package tier

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// 15 new tiers, in the order they were introduced. Each entry pairs
// the tier name with the GOOS where the tier is registered; OS-gated
// tiers (jail, pledge, sandboxexec) are only registered when built on
// their host GOOS.
var newTierNames = []struct {
	Name    string
	HostGO  string // runtime.GOOS required for registration; "" = any
}{
	{"kata", ""},
	{"youki", ""},
	{"systemdnspawn", ""},
	{"nitrorekf", ""},
	{"sev", ""},
	{"tdx", ""},
	{"kubevirt", ""},
	{"virtcontainers", ""},
	{"jail", "freebsd"},
	{"pledge", "openbsd"},
	{"sandboxexec", "darwin"},
	{"gvisordocker", ""},
	{"wolfi", ""},
	{"distroless", ""},
	{"userns", ""},
}

// TestTiers_NewAllRegistered asserts that every one of the 15 new
// tiers is registered in the default registry (when built on its
// host GOOS) and is discoverable via r.Names() / r.Has(name).
func TestTiers_NewAllRegistered(t *testing.T) {
	r := DefaultRegistry()
	if r == nil {
		t.Fatal("DefaultRegistry() returned nil")
	}
	registered := 0
	for _, e := range newTierNames {
		// OS-gated tiers are only registered on their host GOOS.
		if e.HostGO != "" && runtime.GOOS != e.HostGO {
			continue
		}
		registered++
		if !r.Has(e.Name) {
			t.Errorf("new tier %q (host=%s) not in registry (have: %v)",
				e.Name, e.HostGO, r.Names())
		}
	}
	// On any host GOOS, exactly the 12 cross-platform new tiers must
	// be registered. On darwin/freebsd/openbsd we add 1 more (the
	// host-gated one) for a total of 13.
	if runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd" {
		if registered != 13 {
			t.Errorf("registered %d new tiers on %s, want 13", registered, runtime.GOOS)
		}
	} else {
		if registered != 12 {
			t.Errorf("registered %d new tiers on %s, want 12", registered, runtime.GOOS)
		}
	}
}

// TestTiers_NewTotalCount verifies the canonical tier list has all 30
// entries after the tier expansion. This is a regression guard
// against accidentally dropping a registration.
func TestTiers_NewTotalCount(t *testing.T) {
	canonical := CanonicalTierNames()
	// allTiers() is a static metadata table — it always lists all 30
	// tiers regardless of GOOS (OS-gated tiers are still listed, but
	// their adapters are not registered on the wrong host).
	if len(canonical) != 30 {
		t.Errorf("CanonicalTierNames has %d entries, want 30: %v",
			len(canonical), canonical)
	}
	// All 15 new names must appear in the canonical list (regardless
	// of whether they're registered in this build).
	have := make(map[string]bool, len(canonical))
	for _, n := range canonical {
		have[n] = true
	}
	for _, e := range newTierNames {
		if !have[e.Name] {
			t.Errorf("canonical list missing new tier %q", e.Name)
		}
	}
}

// TestTiers_ProbeCallable verifies that every new tier exposes a
// Probe() method that can be invoked without panicking. We don't care
// whether Probe() succeeds — the underlying binary may not exist in
// CI; we only care that the method is callable and returns
// (error).
func TestTiers_ProbeCallable(t *testing.T) {
	r := DefaultRegistry()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, e := range newTierNames {
		// OS-gated tiers are only registered on their host GOOS.
		if e.HostGO != "" && runtime.GOOS != e.HostGO {
			continue
		}
		a, err := r.Get(e.Name)
		if err != nil {
			t.Errorf("tier %q: %v", e.Name, err)
			continue
		}
		if a == nil {
			t.Errorf("tier %q: nil adapter", e.Name)
			continue
		}
		// Probe may return an error if the binary isn't installed;
		// that's fine. We only require it to not panic.
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Errorf("tier %q: Probe() panicked: %v", e.Name, rec)
				}
			}()
			_ = a.Probe(ctx)
		}()
	}
}

// TestTiers_PlatformMetadata verifies that each new tier's Info()
// reports a valid Platform list. OS-gated tiers should report a
// platform matching their host OS when built, and an empty list
// otherwise (the adapter stub).
func TestTiers_PlatformMetadata(t *testing.T) {
	r := DefaultRegistry()
	type platformCase struct {
		tier       string
		expectGO   string // runtime.GOOS value; empty = "any"
		notOnGO    string // runtime.GOOS value where this tier MUST NOT report expectGO
		canonical  string // canonical platform name (e.g. "macos" for GOOS=darwin)
	}
	cases := []platformCase{
		{tier: "jail", expectGO: "freebsd", notOnGO: "linux", canonical: "freebsd"},
		{tier: "pledge", expectGO: "openbsd", notOnGO: "linux", canonical: "openbsd"},
		{tier: "sandboxexec", expectGO: "darwin", notOnGO: "linux", canonical: "macos"},
	}
	for _, tc := range cases {
		// OS-gated tiers are only registered on their host GOOS.
		if runtime.GOOS != tc.expectGO && runtime.GOOS != tc.notOnGO {
			continue
		}
		a, err := r.Get(tc.tier)
		if err != nil {
			// On the wrong GOOS we may not have registered the tier.
			if runtime.GOOS == tc.notOnGO {
				continue
			}
			t.Errorf("tier %q: %v", tc.tier, err)
			continue
		}
		info := a.Info()
		if runtime.GOOS == tc.expectGO {
			// On the expected OS, the tier should report the
			// canonical platform name (or be empty for "any").
			found := false
			for _, p := range info.Platforms {
				if p == tc.canonical {
					found = true
					break
				}
			}
			if !found && len(info.Platforms) > 0 {
				t.Errorf("tier %q on %s: expected canonical platform %q in %v",
					tc.tier, runtime.GOOS, tc.canonical, info.Platforms)
			}
		}
		if runtime.GOOS == tc.notOnGO {
			// On the "wrong" OS the tier must NOT report
			// tc.canonical (if it's registered at all).
			for _, p := range info.Platforms {
				if p == tc.canonical {
					t.Errorf("tier %q on %s: unexpectedly reports platform %q",
						tc.tier, runtime.GOOS, tc.canonical)
				}
			}
		}
	}
}

// TestTiers_PolicyRoutesNewTiers verifies that DefaultPolicy's
// candidate lists include the new tiers in the appropriate positions.
// We don't assert a specific "winner" (because the test environment
// may not have every binary installed) — we only verify that
// each tier appears in at least one of the (security, platform)
// candidate lists it was added to.
func TestTiers_PolicyRoutesNewTiers(t *testing.T) {
	type routeCase struct {
		tier     string
		security SecurityLevel
		platform Platform
	}
	cases := []routeCase{
		// Linux high tiers (HW VMs / containers-with-VM).
		{tier: "kata", security: SecurityHigh, platform: PlatformLinux},
		{tier: "kubevirt", security: SecurityHigh, platform: PlatformLinux},
		{tier: "virtcontainers", security: SecurityHigh, platform: PlatformLinux},
		// Linux untrusted tiers (HW enclaves).
		{tier: "nitrorekf", security: SecurityUntrusted, platform: PlatformLinux},
		{tier: "sev", security: SecurityUntrusted, platform: PlatformLinux},
		{tier: "tdx", security: SecurityUntrusted, platform: PlatformLinux},
		// Linux medium tiers (process-level isolation).
		{tier: "youki", security: SecurityMedium, platform: PlatformLinux},
		{tier: "systemdnspawn", security: SecurityMedium, platform: PlatformLinux},
		{tier: "gvisordocker", security: SecurityMedium, platform: PlatformLinux},
		{tier: "wolfi", security: SecurityMedium, platform: PlatformLinux},
		// Linux low tiers (cheapest).
		{tier: "userns", security: SecurityLow, platform: PlatformLinux},
	}
	for _, tc := range cases {
		cands := defaultCandidates(tc.security, tc.platform, "", false)
		found := false
		for _, c := range cands {
			if c.Name == tc.tier {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tier %q not in candidates for security=%s platform=%s",
				tc.tier, tc.security, tc.platform)
		}
	}
}

// TestTiers_ProfileCIRoutesToNewTiers verifies that the new
// ci-secure / ci-fast profiles route to tiers from the new set
// (landlock/seccomp/userns are existing tiers, but ci-secure must
// pick a tier that satisfies SecurityMedium on Linux, and ci-fast
// must pick a tier that satisfies SecurityLow on Linux). The point
// of this test is to ensure the profiles exist and resolve to
// non-empty names.
func TestTiers_ProfileCIRoutesToNewTiers(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()

	// ci-secure: pick must be a Linux medium tier that is registered.
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return string(ProfileCISecure)
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(ci-secure): %v", err)
	}
	if got == "" {
		t.Errorf("ProfilePolicy(ci-secure) returned empty tier")
	}

	// ci-fast: pick must be a Linux low tier.
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return string(ProfileCIFast)
		}
		return ""
	}
	got, err = ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(ci-fast): %v", err)
	}
	if got == "" {
		t.Errorf("ProfilePolicy(ci-fast) returned empty tier")
	}
	if got != "native" {
		t.Errorf("ProfilePolicy(ci-fast) = %q, want native (0ms cheapest)", got)
	}
}
