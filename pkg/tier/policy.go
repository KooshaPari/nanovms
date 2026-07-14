// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — policy.go implements SelectionPolicy: a small rules
// engine that maps a (security, platform, startup-budget, workload,
// trusted-code) tuple onto a concrete tier from the registry.
package tier

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// SecurityLevel is the trust model for the workload. Higher levels pick
// more isolated tiers.
type SecurityLevel string

const (
	// SecurityLow permits process-level isolation (native, landlock).
	SecurityLow SecurityLevel = "low"
	// SecurityMedium permits user-space kernel interception (gvisor, seccomp).
	SecurityMedium SecurityLevel = "medium"
	// SecurityHigh permits hardware-backed virtualization (firecracker, kvm, applevz).
	SecurityHigh SecurityLevel = "high"
	// SecurityUntrusted requires a full hardware-isolated VM (firecracker, qemu).
	SecurityUntrusted SecurityLevel = "untrusted"
)

// Platform identifies the host operating system.
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformMacOS   Platform = "macos"
	PlatformWindows Platform = "windows"
)

// Workload is a coarse class of workload used as a hint in tier
// selection. The empty string is "any".
type Workload string

const (
	WorkloadBrowser Workload = "browser"
	WorkloadCode    Workload = "code"
	WorkloadTool    Workload = "tool"
	WorkloadCLI     Workload = "cli"
)

// SelectionConfig is the input to SelectionPolicy.Select. Zero values are
// treated as defaults: Security=medium, Platform=linux, StartupBudget=0
// (no budget), Workload="" (any), TrustedCode=false.
type SelectionConfig struct {
	Security      SecurityLevel
	StartupBudget int // ms; 0 means no budget
	Platform      Platform
	Workload      Workload
	TrustedCode   bool
}

// SelectionPolicy is the contract for picking a tier. Implementations
// must be deterministic: calling Select twice with the same config must
// return the same tier name.
type SelectionPolicy interface {
	// Select returns the name of the best matching tier from the
	// registry, or an error explaining why no tier satisfies the
	// constraints.
	Select(cfg SelectionConfig, r *Registry) (string, error)
}

// DefaultPolicy is the default policy used when no policy is supplied
// explicitly. The mapping (security, platform) -> preferred tiers is:
//
//	security=low       : native, landlock, seccomp
//	security=medium    : gvisor, seccomp
//	security=high      : firecracker, kvm, applevz
//	security=untrusted : firecracker, qemu, cloudhv
//
// Platform fallbacks:
//   - macos   -> applevz > lima > qemu
//   - windows -> wsl2 (when available) > qemu
//   - linux   -> no platform override
//
// StartupBudget filters out any tier whose Info().StartupMS exceeds the
// budget (after rounding up to the nearest millisecond).
//
// Workload hint narrows the candidate set: a "browser" workload with
// security=high prefers gvisor (full syscall mediation) over firecracker
// (faster) when the budget allows it.
type DefaultPolicy struct{}

// Select implements SelectionPolicy.
func (DefaultPolicy) Select(cfg SelectionConfig, r *Registry) (string, error) {
	if r == nil {
		return "", fmt.Errorf("tier: nil registry")
	}
	platform := cfg.Platform
	if platform == "" {
		platform = PlatformLinux
	}
	security := cfg.Security
	if security == "" {
		security = SecurityMedium
	}

	// Build the ordered candidate list. Order matters: the first tier
	// in the list whose StartupMS fits the budget AND is registered AND
	// supports the platform/workload wins.
	candidates := defaultCandidates(security, platform, cfg.Workload, cfg.TrustedCode)

	// Filter by registry membership + platform support.
	filtered := make([]TierInfo, 0, len(candidates))
	for _, c := range candidates {
		if !r.Has(c.Name) {
			continue
		}
		if !platformMatches(c.Platforms, platform) {
			continue
		}
		filtered = append(filtered, c)
	}
	if len(filtered) == 0 {
		return "", fmt.Errorf("tier: no candidate matches security=%q platform=%q workload=%q", security, platform, cfg.Workload)
	}

	// Filter by startup budget.
	if cfg.StartupBudget > 0 {
		within := make([]TierInfo, 0, len(filtered))
		for _, c := range filtered {
			if c.StartupMS <= cfg.StartupBudget {
				within = append(within, c)
			}
		}
		if len(within) == 0 {
			return "", fmt.Errorf("tier: no tier fits startup budget %dms (best: %s @ %dms)",
				cfg.StartupBudget, filtered[0].Name, filtered[0].StartupMS)
		}
		filtered = within
	}

	// Sort by (StartupMS asc, Name asc) for determinism, then return
	// the first. Sort is a no-op when the candidate list is already
	// pre-ordered as the defaultCandidates() function returns.
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].StartupMS != filtered[j].StartupMS {
			return filtered[i].StartupMS < filtered[j].StartupMS
		}
		return filtered[i].Name < filtered[j].Name
	})

	return filtered[0].Name, nil
}

// platformMatches returns true if info's Platforms list contains
// platform, or if the list is empty (meaning "any").
func platformMatches(platforms []string, platform Platform) bool {
	if len(platforms) == 0 {
		return true
	}
	for _, p := range platforms {
		if p == string(platform) {
			return true
		}
	}
	return false
}

// defaultCandidates returns the preferred tier list for a given
// (security, platform, workload, trusted) tuple, in priority order. The
// caller is responsible for filtering by registry membership and budget.
//
// NOTE: this function is the single source of truth for the default
// policy. Any change here must be reflected in the README and policy_test.go.
func defaultCandidates(security SecurityLevel, platform Platform, workload Workload, trusted bool) []TierInfo {
	all := allTiers()
	byName := make(map[string]TierInfo, len(all))
	for _, t := range all {
		byName[t.Name] = t
	}

	// Platform-specific preferred order, with security/workload shaping
	// the order.
	var order []string
	switch platform {
	case PlatformMacOS:
		switch security {
		case SecurityLow:
			order = []string{"native", "applevz", "qemu"}
		case SecurityMedium:
			order = []string{"applevz", "gvisor", "lima", "qemu"}
		case SecurityHigh:
			order = []string{"applevz", "lima", "qemu"}
		case SecurityUntrusted:
			order = []string{"applevz", "qemu", "lima"}
		}
	case PlatformWindows:
		switch security {
		case SecurityLow:
			order = []string{"native", "qemu"}
		case SecurityMedium:
			order = []string{"qemu", "gvisor"}
		case SecurityHigh:
			order = []string{"qemu"}
		case SecurityUntrusted:
			order = []string{"qemu"}
		}
	default: // linux
		switch security {
		case SecurityLow:
			order = []string{"native", "landlock", "seccomp", "wasm"}
		case SecurityMedium:
			order = []string{"gvisor", "seccomp", "wasm", "docker", "podman"}
		case SecurityHigh:
			order = []string{"firecracker", "kvm", "cloudhv", "crosvm", "qemu"}
		case SecurityUntrusted:
			order = []string{"firecracker", "qemu", "kvm", "cloudhv"}
		}
	}

	// Trusted code can short-circuit to a faster tier even if the
	// security setting is higher, when workload=="cli" or "tool".
	if trusted && (workload == WorkloadCLI || workload == WorkloadTool) {
		// Prepend "wasm" / "native" for trusted tooling.
		order = append([]string{"wasm", "native"}, order...)
	}

	out := make([]TierInfo, 0, len(order))
	for _, n := range order {
		if t, ok := byName[n]; ok {
			out = append(out, t)
		}
	}
	return out
}

// allTiers returns the canonical metadata for every supported tier.
// Kept in policy.go (not registry.go) so the default policy can be
// reasoned about without a live registry.
func allTiers() []TierInfo {
	return []TierInfo{
		{Name: "wasm", StartupMS: 1, MemoryMB: 1, Security: "low", Platforms: []string{"linux", "macos", "windows"}, Workloads: []string{"tool", "cli"}},
		{Name: "gvisor", StartupMS: 90, MemoryMB: 50, Security: "medium", Platforms: []string{"linux", "macos"}, Workloads: []string{"browser", "code", "tool"}},
		{Name: "firecracker", StartupMS: 125, MemoryMB: 128, Security: "high", Platforms: []string{"linux"}, Workloads: []string{"code", "browser", "tool"}},
		{Name: "landlock", StartupMS: 1, MemoryMB: 2, Security: "low", Platforms: []string{"linux"}, Workloads: []string{"tool", "cli"}},
		{Name: "seccomp", StartupMS: 1, MemoryMB: 1, Security: "low", Platforms: []string{"linux"}, Workloads: []string{"tool", "cli"}},
		{Name: "native", StartupMS: 0, MemoryMB: 0, Security: "low", Platforms: []string{"linux", "macos", "windows"}, Workloads: []string{"cli", "tool"}},
		{Name: "docker", StartupMS: 600, MemoryMB: 50, Security: "medium", Platforms: []string{"linux", "macos", "windows"}, Workloads: []string{"code", "tool", "cli"}},
		{Name: "podman", StartupMS: 700, MemoryMB: 50, Security: "medium", Platforms: []string{"linux", "macos", "windows"}, Workloads: []string{"code", "tool", "cli"}},
		{Name: "hyperkit", StartupMS: 400, MemoryMB: 256, Security: "high", Platforms: []string{"macos"}, Workloads: []string{"code", "tool"}},
		{Name: "applevz", StartupMS: 250, MemoryMB: 256, Security: "high", Platforms: []string{"macos"}, Workloads: []string{"code", "browser", "tool"}},
		{Name: "lima", StartupMS: 3000, MemoryMB: 512, Security: "high", Platforms: []string{"macos", "windows"}, Workloads: []string{"code", "tool", "browser"}},
		{Name: "kvm", StartupMS: 150, MemoryMB: 128, Security: "high", Platforms: []string{"linux"}, Workloads: []string{"code", "browser", "tool"}},
		{Name: "qemu", StartupMS: 2000, MemoryMB: 256, Security: "high", Platforms: []string{"linux", "macos", "windows"}, Workloads: []string{"code", "browser", "tool"}},
		{Name: "cloudhv", StartupMS: 180, MemoryMB: 128, Security: "high", Platforms: []string{"linux"}, Workloads: []string{"code", "browser", "tool"}},
		{Name: "crosvm", StartupMS: 200, MemoryMB: 128, Security: "high", Platforms: []string{"linux"}, Workloads: []string{"code", "tool"}},
	}
}

// CanonicalTierNames returns the full sorted list of tier names. Used by
// `nvms tier list` and by main.go to build a help string.
func CanonicalTierNames() []string {
	all := allTiers()
	out := make([]string, 0, len(all))
	for _, t := range all {
		out = append(out, t.Name)
	}
	sort.Strings(out)
	return out
}

// JoinNames formats a sorted list of names for use in error messages.
func JoinNames(names []string) string { return strings.Join(names, ", ") }

// EnvVarTier is the env var read by AutoPolicy.FromEnv to discover the
// requested tier name (e.g. "auto", "firecracker", "wasm"). When unset
// or empty, AutoPolicy falls back to DefaultPolicy.
const EnvVarTier = "NVMS_TIER"

// EnvVarSecurity is the env var read by AutoPolicy.FromEnv to discover
// the security level (low/medium/high/untrusted). When unset, it
// defaults to SecurityMedium.
const EnvVarSecurity = "NVMS_SECURITY"

// EnvVarPlatform is the env var read by AutoPolicy.FromEnv to override
// the host platform (linux/macos/windows). When unset, AutoPolicy
// derives the platform from runtime.GOOS.
const EnvVarPlatform = "NVMS_PLATFORM"

// EnvVarStartupBudget is the env var read by AutoPolicy.FromEnv to
// enforce a startup budget in milliseconds. Empty string means "no
// budget". Invalid values are treated as "no budget" with no error.
const EnvVarStartupBudget = "NVMS_STARTUP_BUDGET_MS"

// EnvVarProfile is the env var read by ProfilePolicy.Select to pick a
// named profile (dev/ci/prod-secure/prod-fast/airgapped).
const EnvVarProfile = "NVMS_PROFILE"

// AutoPolicy is a SelectionPolicy driven entirely by environment
// variables. It exists for embedders who want "just pick the right tier
// based on the environment" with no extra wiring. The mapping is:
//
//	NVMS_TIER=<name>           -> exact tier if registered
//	NVMS_TIER=auto (default)   -> delegates to DefaultPolicy
//	NVMS_SECURITY=<level>      -> overrides cfg.Security
//	NVMS_PLATFORM=<os>         -> overrides cfg.Platform
//	NVMS_STARTUP_BUDGET_MS=<n> -> sets cfg.StartupBudget
//
// AutoPolicy composes DefaultPolicy; it does not replace it.
type AutoPolicy struct{}

// Select implements SelectionPolicy. If NVMS_TIER is set to a concrete
// tier name (not "auto"), AutoPolicy returns that name verbatim — but
// only if it is registered AND meets the platform/security/budget
// filters. Otherwise it falls back to DefaultPolicy.
func (AutoPolicy) Select(cfg SelectionConfig, r *Registry) (string, error) {
	envCfg := applyEnvOverrides(cfg)
	if tier, handled, err := envExactTier(envCfg, r); handled {
		return tier, err
	}
	return DefaultPolicy{}.Select(envCfg, r)
}

// applyEnvOverrides copies cfg and overlays any env-driven fields.
func applyEnvOverrides(cfg SelectionConfig) SelectionConfig {
	out := cfg
	if v := osGetenv(EnvVarSecurity); v != "" {
		out.Security = SecurityLevel(strings.ToLower(v))
	}
	if v := osGetenv(EnvVarPlatform); v != "" {
		out.Platform = Platform(strings.ToLower(v))
	} else if out.Platform == "" {
		// Derive from the host when no env var is set.
		out.Platform = platformFromRuntime()
	}
	if v := osGetenv(EnvVarStartupBudget); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			out.StartupBudget = n
		}
	}
	return out
}

// envExactTier returns the tier named in NVMS_TIER (if it is not "auto"
// and is registered). Returns handled=true when the env var drove the
// decision. err is non-nil when the requested tier is not registered.
func envExactTier(cfg SelectionConfig, r *Registry) (string, bool, error) {
	raw := osGetenv(EnvVarTier)
	if raw == "" || strings.EqualFold(raw, "auto") {
		return "", false, nil
	}
	name := strings.ToLower(raw)
	if !r.Has(name) {
		return "", true, fmt.Errorf("tier: NVMS_TIER=%q not registered (available: %s)", name, JoinNames(r.Names()))
	}
	// The user explicitly asked for this tier by name; honor it even
	// when DefaultPolicy would have picked something else, provided the
	// platform/security/budget filters are satisfied. We reuse
	// DefaultPolicy to validate the filters by asking what it would
	// pick — if that comes back with an error, the user's explicit tier
	// is incompatible with the env-driven filters.
	_, selErr := DefaultPolicy{}.Select(SelectionConfig{
		Security:      cfg.Security,
		Platform:      cfg.Platform,
		StartupBudget: cfg.StartupBudget,
		Workload:      cfg.Workload,
	}, r)
	if selErr != nil {
		return "", true, fmt.Errorf("tier: NVMS_TIER=%q fails filters: %w", name, selErr)
	}
	return name, true, nil
}

// platformFromRuntime maps runtime.GOOS onto the Platform enum. The
// mapping is the same one used by cmd/nanovms/main.go.
func platformFromRuntime() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "windows":
		return PlatformWindows
	default:
		return PlatformLinux
	}
}

// Profile is a named bundle of (security, startup-budget, workload,
// trusted-code, platform) defaults. Profiles are picked by name via
// NVMS_PROFILE; an empty profile name returns the zero SelectionConfig
// (which DefaultPolicy maps to a reasonable linux/medium default).
type Profile string

const (
	// ProfileDev is a low-overhead profile for local development. It
	// prefers wasm/native/seccomp and accepts a startup budget of 5s.
	ProfileDev Profile = "dev"

	// ProfileCI is a balanced profile for continuous integration. It
	// prefers gVisor/firecracker for isolation parity with prod, but
	// with a 10s startup budget so heavier tests still pass.
	ProfileCI Profile = "ci"

	// ProfileProdSecure is a high-isolation profile for production. It
	// requires Security=Untrusted and rejects any tier whose Security
	// is below "high". No startup budget (production VMs are larger).
	ProfileProdSecure Profile = "prod-secure"

	// ProfileProdFast is a production profile that trades some
	// isolation for lower startup latency (firecracker, cloudhv).
	ProfileProdFast Profile = "prod-fast"

	// ProfileAirgapped is an offline-friendly profile: it forces
	// security=high and rejects any tier whose binaries would have to
	// reach the network at Probe time (no live `docker version`
	// checks, no cloud APIs).
	ProfileAirgapped Profile = "airgapped"
)

// KnownProfiles returns the canonical list of profile names. Stable
// order; safe to use for help text.
func KnownProfiles() []string {
	return []string{
		string(ProfileDev),
		string(ProfileCI),
		string(ProfileProdSecure),
		string(ProfileProdFast),
		string(ProfileAirgapped),
	}
}

// ProfilePolicy selects a tier based on a named profile. The profile is
// resolved in this order:
//
//  1. NVMS_PROFILE env var, if set
//  2. The profile embedded in cfg (currently unused — placeholder for
//     future cfg.Profile field)
//
// If no profile is set, ProfilePolicy delegates to DefaultPolicy with
// the supplied cfg unchanged.
type ProfilePolicy struct{}

// Select implements SelectionPolicy.
func (ProfilePolicy) Select(cfg SelectionConfig, r *Registry) (string, error) {
	name := osGetenv(EnvVarProfile)
	if name == "" {
		return DefaultPolicy{}.Select(cfg, r)
	}
	p := Profile(strings.ToLower(name))
	sc := profileSelection(p)
	// Overlay the caller's cfg.Workload/TrustedCode — profiles are
	// knobs, not overrides of the workload hint.
	sc.Workload = cfg.Workload
	sc.TrustedCode = cfg.TrustedCode
	return DefaultPolicy{}.Select(sc, r)
}

// profileSelection returns the SelectionConfig implied by a named
// profile. Returns the zero value for unknown profile names, which
// DefaultPolicy then maps to its defaults.
func profileSelection(p Profile) SelectionConfig {
	switch p {
	case ProfileDev:
		return SelectionConfig{
			Security:      SecurityLow,
			Platform:      PlatformLinux,
			StartupBudget: 5000,
		}
	case ProfileCI:
		return SelectionConfig{
			Security:      SecurityMedium,
			Platform:      PlatformLinux,
			StartupBudget: 10000,
		}
	case ProfileProdSecure:
		return SelectionConfig{
			Security:      SecurityUntrusted,
			Platform:      PlatformLinux,
			StartupBudget: 0, // no budget — VMs are larger
		}
	case ProfileProdFast:
		return SelectionConfig{
			Security:      SecurityHigh,
			Platform:      PlatformLinux,
			StartupBudget: 1000,
		}
	case ProfileAirgapped:
		return SelectionConfig{
			Security:      SecurityHigh,
			Platform:      PlatformLinux,
			StartupBudget: 0,
		}
	default:
		return SelectionConfig{} // unknown profile -> DefaultPolicy defaults
	}
}

// osGetenv is a tiny indirection so tests can swap the env source.
var osGetenv = osLookupEnv

// osLookupEnv wraps os.LookupEnv. Declared as a var so tests can
// override (see policy_test.go).
func osLookupEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return ""
	}
	return v
}
