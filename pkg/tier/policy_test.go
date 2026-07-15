// SPDX-License-Identifier: MIT OR Apache-2.0
package tier

import (
	"strings"
	"testing"
)

// TestPolicy_DefaultLinuxMedium verifies linux + security=medium.
// After tier expansion the linux+medium candidate order begins with
// [landlock, seccomp, userns, gvisor, ...]; deterministic (StartupMS
// asc, Name asc) sort picks landlock (1ms, alphabetical first among
// 1ms tiers).
func TestPolicy_DefaultLinuxMedium(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityMedium,
		Platform: PlatformLinux,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "landlock" {
		t.Fatalf("linux+medium = %q, want landlock (cheapest startup among preferred tiers)", got)
	}
}

// TestPolicy_DefaultLinuxHigh verifies linux + security=high.
// After tier expansion the high-security Linux candidate list includes
// distroless (80ms), which now beats firecracker (125ms) and is the
// fastest high-security Linux tier.
func TestPolicy_DefaultLinuxHigh(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityHigh,
		Platform: PlatformLinux,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "distroless" {
		t.Fatalf("linux+high = %q, want distroless (cheapest high-security Linux tier)", got)
	}
}

// TestPolicy_DefaultMacOSMedium verifies macos + security=medium.
// The macos+medium candidate list now includes distroless (80ms) ahead
// of gvisor (90ms), so distroless wins.
func TestPolicy_DefaultMacOSMedium(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityMedium,
		Platform: PlatformMacOS,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "distroless" {
		t.Fatalf("macos+medium = %q, want distroless", got)
	}
}

// TestPolicy_DefaultWindowsMedium verifies windows + security=medium.
// The windows+medium candidate list now includes distroless (80ms)
// ahead of qemu (2000ms), so distroless wins.
func TestPolicy_DefaultWindowsMedium(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityMedium,
		Platform: PlatformWindows,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "distroless" {
		t.Fatalf("windows+medium = %q, want distroless", got)
	}
}

// TestPolicy_StartupBudget_FailsWhenUnsatisfiable verifies that a
// tight budget that excludes every candidate returns a clear error
// mentioning the budget value. Note: with the new tiers the linux+high
// candidate list now includes distroless (80ms) and virtcontainers
// (200ms), so we drop the budget to 50ms to truly exclude everything.
func TestPolicy_StartupBudget_FailsWhenUnsatisfiable(t *testing.T) {
	_, err := DefaultPolicy{}.Select(SelectionConfig{
		Security:      SecurityHigh,
		Platform:      PlatformLinux,
		StartupBudget: 50,
	}, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected budget-exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "50ms") {
		t.Fatalf("error should mention 50ms budget: %v", err)
	}
}

// TestPolicy_StartupBudget_PicksFastEnough verifies that a 200ms
// budget on linux/medium is satisfiable. The candidate order begins
// [landlock, seccomp, userns, ...] and deterministic sort picks
// landlock (1ms, alphabetically first among 1ms tiers).
func TestPolicy_StartupBudget_PicksFastEnough(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security:      SecurityMedium,
		Platform:      PlatformLinux,
		StartupBudget: 200,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "landlock" {
		t.Fatalf("linux+medium+budget200 = %q, want landlock", got)
	}
}

// TestPolicy_WorkloadHint_TrustedCLI verifies that trusted CLI/tool
// workloads prepend wasm/native, which then dominates via the
// startup-MIN sort (native=0ms wins).
func TestPolicy_WorkloadHint_TrustedCLI(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security:    SecurityMedium,
		Platform:    PlatformLinux,
		Workload:    WorkloadCLI,
		TrustedCode: true,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "native" {
		t.Fatalf("trusted cli on linux/medium = %q, want native (0ms startup)", got)
	}
}

// TestPolicy_NilRegistryRejected verifies Select with a nil registry
// returns a clear error rather than panicking.
func TestPolicy_NilRegistryRejected(t *testing.T) {
	_, err := DefaultPolicy{}.Select(SelectionConfig{}, nil)
	if err == nil {
		t.Fatalf("expected error for nil registry")
	}
}

// TestPolicy_DefaultsWhenEmpty verifies that an empty SelectionConfig
// resolves to a non-empty tier (linux+medium defaults) without panicking.
func TestPolicy_DefaultsWhenEmpty(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got == "" {
		t.Fatalf("empty cfg returned empty tier")
	}
}

// TestPolicy_AutoEnvOverlay verifies AutoPolicy reads NVMS_TIER and
// returns the exact tier name verbatim.
func TestPolicy_AutoEnvOverlay(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		switch key {
		case EnvVarTier:
			return "firecracker"
		case EnvVarSecurity:
			return "high"
		}
		return ""
	}
	got, err := AutoPolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("AutoPolicy: %v", err)
	}
	if got != "firecracker" {
		t.Fatalf("AutoPolicy with NVMS_TIER=firecracker = %q", got)
	}
}

// TestPolicy_AutoEnvUnknown verifies AutoPolicy returns an error
// when NVMS_TIER is set to an unregistered name.
func TestPolicy_AutoEnvUnknown(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		switch key {
		case EnvVarTier:
			return "ghost"
		}
		return ""
	}
	_, err := AutoPolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected error for unknown NVMS_TIER")
	}
}

// TestPolicy_AutoAutoDefersToDefault verifies that when NVMS_TIER=auto
// (or unset), AutoPolicy falls back to DefaultPolicy. After tier
// expansion, linux+medium picks landlock.
func TestPolicy_AutoAutoDefersToDefault(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(string) string { return "" }

	got, err := AutoPolicy{}.Select(SelectionConfig{
		Security: SecurityMedium,
		Platform: PlatformLinux,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("AutoPolicy: %v", err)
	}
	if got != "landlock" {
		t.Fatalf("default policy = %q, want landlock (linux+medium pick after expansion)", got)
	}
}

// TestPolicy_ProfilePolicy_Dev exercises ProfilePolicy with NVMS_PROFILE=dev.
// dev implies SecurityLow, StartupBudget=5000; "native" (0ms) wins.
func TestPolicy_ProfilePolicy_Dev(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return "dev"
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(dev): %v", err)
	}
	if got != "native" {
		t.Fatalf("ProfilePolicy(dev) = %q, want native", got)
	}
}

// TestPolicy_ProfilePolicy_ProdSecure exercises NVMS_PROFILE=prod-secure
// and verifies it returns firecracker (linux, security=untrusted,
// no budget).
func TestPolicy_ProfilePolicy_ProdSecure(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return "prod-secure"
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(prod-secure): %v", err)
	}
	if got != "firecracker" {
		t.Fatalf("ProfilePolicy(prod-secure) = %q, want firecracker", got)
	}
}

// TestPolicy_ProfilePolicy_ProdFast exercises NVMS_PROFILE=prod-fast.
// prod-fast implies SecurityHigh, StartupBudget=1000ms; the cheapest
// high-security Linux tier after tier expansion is distroless (80ms).
func TestPolicy_ProfilePolicy_ProdFast(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return "prod-fast"
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(prod-fast): %v", err)
	}
	if got != "distroless" {
		t.Fatalf("ProfilePolicy(prod-fast) = %q, want distroless", got)
	}
}

// TestPolicy_ProfilePolicy_CI exercises NVMS_PROFILE=ci.
// ci implies SecurityMedium, StartupBudget=10000ms; the cheapest
// medium-security Linux tier after tier expansion is landlock (1ms,
// alphabetically first among 1ms tiers).
func TestPolicy_ProfilePolicy_CI(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return "ci"
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(ci): %v", err)
	}
	if got != "landlock" {
		t.Fatalf("ProfilePolicy(ci) = %q, want landlock", got)
	}
}

// TestPolicy_ProfilePolicy_Airgapped exercises NVMS_PROFILE=airgapped.
// airgapped implies SecurityHigh on Linux, no budget. After tier
// expansion the linux+high candidate list includes distroless (80ms),
// which now wins.
func TestPolicy_ProfilePolicy_Airgapped(t *testing.T) {
	prev := osGetenv
	defer func() { osGetenv = prev }()
	osGetenv = func(key string) string {
		if key == EnvVarProfile {
			return "airgapped"
		}
		return ""
	}
	got, err := ProfilePolicy{}.Select(SelectionConfig{}, DefaultRegistry())
	if err != nil {
		t.Fatalf("ProfilePolicy(airgapped): %v", err)
	}
	if got != "distroless" {
		t.Fatalf("ProfilePolicy(airgapped) = %q, want distroless", got)
	}
}

// TestPolicy_KnownProfiles verifies KnownProfiles() returns the named
// profiles in stable order for use in help text. After expansion we
// include the new ci-secure / ci-fast profiles.
func TestPolicy_KnownProfiles(t *testing.T) {
	got := KnownProfiles()
	want := []string{"dev", "ci", "ci-secure", "ci-fast", "prod-secure", "prod-fast", "airgapped"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("KnownProfiles() = %v, want %v", got, want)
	}
}

// TestPolicy_LinuxLowNativeWins verifies that linux+low picks the
// cheapest (0ms) tier: native.
func TestPolicy_LinuxLowNativeWins(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityLow,
		Platform: PlatformLinux,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "native" {
		t.Fatalf("linux+low = %q, want native", got)
	}
}

// TestPolicy_LinuxUntrustedFirecracker verifies that linux+untrusted
// (highest trust requirement) lands on firecracker.
func TestPolicy_LinuxUntrustedFirecracker(t *testing.T) {
	got, err := DefaultPolicy{}.Select(SelectionConfig{
		Security: SecurityUntrusted,
		Platform: PlatformLinux,
	}, DefaultRegistry())
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got != "firecracker" {
		t.Fatalf("linux+untrusted = %q, want firecracker", got)
	}
}

// TestPolicy_CanonicalTierNames verifies the canonical sorted list of
// all 15 tier names is exposed for help text and CLI surfaces.
func TestPolicy_CanonicalTierNames(t *testing.T) {
	names := CanonicalTierNames()
	if len(names) < 15 {
		t.Fatalf("expected >=15 canonical names, got %d (%v)", len(names), names)
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("CanonicalTierNames not sorted at %d: %v", i, names)
		}
	}
}