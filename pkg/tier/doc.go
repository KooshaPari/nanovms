// Package tier provides a pluggable registry of sandbox-isolation backends
// (Firecracker, gVisor, WASM, KVM, QEMU, AppleVZ, ...) and the rules engine
// that picks one for a given (security, platform, startup-budget, workload)
// tuple.
//
// # Adapter contract
//
// Every tier implements Adapter — Deploy / Start / Stop / Delete /
// GetStartupTime / Info / Probe. The Probe method must not start any
// workload; it only inspects the host (binary on $PATH, /dev/kvm,
// sysctl kern.hv_vmm_present, ...) and returns nil when the runtime is
// usable.
//
// # Registry
//
// The registry is keyed by lowercase canonical name ("firecracker",
// "gvisor", "wasm", "landlock", "seccomp", "native", "qemu", "kvm",
// "cloudhv", "crosvm", "applevz", "hyperkit", "lima", "docker",
// "podman"). DefaultRegistry() returns a registry pre-populated with all
// 15 tiers; callers may also construct an empty registry and use
// Register / MustRegister / RegisterAll to wire a custom subset.
//
// # Selection policy
//
// DefaultPolicy picks a tier deterministically from the canonical
// metadata table in policy.go; AutoPolicy overlays NVMS_TIER /
// NVMS_SECURITY / NVMS_PLATFORM / NVMS_STARTUP_BUDGET_MS env vars on top
// of the default; ProfilePolicy is a named-profile knob (dev / ci /
// prod-secure / prod-fast / airgapped). All three satisfy
// SelectionPolicy and are safe to substitute.
package tier