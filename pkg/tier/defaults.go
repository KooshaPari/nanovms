// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — defaults.go wires the full set of 15+ tiers into the
// default registry.
package tier

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// baseAdapter is a small helper struct that tier adapters can embed to
// inherit the Info() implementation. Adapters that need a more elaborate
// description or workload hint can override Info() directly.
//
// It also provides a default Probe() that always succeeds; concrete
// adapters override it to actually inspect the host.
type baseAdapter struct {
	info TierInfo
}

// Info returns the static metadata. Provided so all tier adapters that
// embed *baseAdapter automatically satisfy Adapter.Info().
func (b *baseAdapter) Info() TierInfo { return b.info }

// defaultProbe always succeeds. Concrete adapters should override this
// with a real check (kernel feature, $PATH lookup, etc.).
func (b *baseAdapter) defaultProbe() error { return nil }

// newSandbox is a small constructor used by every tier's Deploy method to
// keep the ID format and field population consistent. It deliberately does
// NOT call Start — that's the adapter's responsibility.
func newSandbox(tier, name string, kind domain.SandboxType, flavor domain.VMFlavor, config *domain.SandboxConfig) *domain.Sandbox {
	return &domain.Sandbox{
		ID:        fmt.Sprintf("%s-%s", tier, domain.GenerateID()),
		Name:      name,
		Status:    domain.SandboxStatusRunning,
		Type:      kind,
		VMFlavor:  flavor,
		Config:    config,
		CreatedAt: time.Now(),
	}
}

// noopLifecycle provides no-op Start/Stop/Delete implementations suitable
// for lightweight tiers (e.g. native, seccomp) where lifecycle is implicit
// in the calling process. Tiers that need a real process to manage should
// override these.
type noopLifecycle struct{}

// Start is a no-op for tiers where lifecycle is implicit.
func (noopLifecycle) Start(_ context.Context, _ string) error { return nil }

// Stop is a no-op for tiers where lifecycle is implicit.
func (noopLifecycle) Stop(_ context.Context, _ string) error { return nil }

// Delete is a no-op for tiers where lifecycle is implicit.
func (noopLifecycle) Delete(_ context.Context, _ string) error { return nil }

// sortStrings is a small helper used by tier metadata lists so the output
// of `nvms tier info` is deterministic.
func sortStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// registerDefaultTiers populates r with the canonical set of 15 tiers.
// Order is preserved within Info() but not relevant for the registry map.
func registerDefaultTiers(r *Registry) {
	// Tier 1-3 (original trio, preserved as-is).
	r.MustRegister("wasm", NewWASMAdapter())
	r.MustRegister("gvisor", NewGVisorAdapter())
	r.MustRegister("firecracker", NewFirecrackerAdapter())

	// Tier 4-6: process-level isolation primitives.
	r.MustRegister("landlock", NewLandlockAdapter())
	r.MustRegister("seccomp", NewSeccompAdapter())
	r.MustRegister("native", NewNativeAdapter())

	// Tier 7-8: OCI container runtimes.
	r.MustRegister("docker", NewDockerAdapter())
	r.MustRegister("podman", NewPodmanAdapter())

	// Tier 9-11: macOS-targeted VM tiers.
	r.MustRegister("hyperkit", NewHyperKitAdapter())
	r.MustRegister("applevz", NewAppleVZAdapter())
	r.MustRegister("lima", NewLimaAdapter())

	// Tier 12-15: Linux hypervisors and full-system emulators.
	r.MustRegister("kvm", NewKVMAdapter())
	r.MustRegister("qemu", NewQEMUAdapter())
	r.MustRegister("cloudhv", NewCloudHypervisorAdapter())
	r.MustRegister("crosvm", NewCrosvmAdapter())
}
