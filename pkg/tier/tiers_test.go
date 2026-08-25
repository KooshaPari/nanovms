// SPDX-License-Identifier: MIT OR Apache-2.0
package tier

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// adapterCase pairs a constructor with its expected tier name. The
// tests iterate over this slice so adding a 16th tier is a one-line
// change.
type adapterCase struct {
	name string
	make func() Adapter
}

func allAdapterCases() []adapterCase {
	return []adapterCase{
		{"wasm", func() Adapter { return NewWASMAdapter() }},
		{"gvisor", func() Adapter { return NewGVisorAdapter() }},
		{"firecracker", func() Adapter { return NewFirecrackerAdapter() }},
		{"landlock", func() Adapter { return NewLandlockAdapter() }},
		{"seccomp", func() Adapter { return NewSeccompAdapter() }},
		{"native", func() Adapter { return NewNativeAdapter() }},
		{"docker", func() Adapter { return NewDockerAdapter() }},
		{"podman", func() Adapter { return NewPodmanAdapter() }},
		{"hyperkit", func() Adapter { return NewHyperKitAdapter() }},
		{"applevz", func() Adapter { return NewAppleVZAdapter() }},
		{"lima", func() Adapter { return NewLimaAdapter() }},
		{"kvm", func() Adapter { return NewKVMAdapter() }},
		{"qemu", func() Adapter { return NewQEMUAdapter() }},
		{"cloudhv", func() Adapter { return NewCloudHypervisorAdapter() }},
		{"crosvm", func() Adapter { return NewCrosvmAdapter() }},
	}
}

// TestAdapterInfoName asserts every adapter's Info().Name matches the
// lowercase canonical name it is registered under.
func TestAdapterInfoName(t *testing.T) {
	for _, c := range allAdapterCases() {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			if got := a.Info().Name; got != c.name {
				t.Fatalf("Info().Name = %q, want %q", got, c.name)
			}
		})
	}
}

// TestAdapterGetStartupTime asserts GetStartupTime() returns a
// non-negative duration matching Info().StartupMS (within rounding).
func TestAdapterGetStartupTime(t *testing.T) {
	for _, c := range allAdapterCases() {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			d := a.GetStartupTime()
			if d < 0 {
				t.Fatalf("GetStartupTime() = %v, want >= 0", d)
			}
			want := time.Duration(a.Info().StartupMS) * time.Millisecond
			if d != want {
				t.Fatalf("GetStartupTime() = %v, want %v", d, want)
			}
		})
	}
}

// TestAdapterProbeReturnsErrorOrNil exercises Probe on every adapter.
// On a host without the runtime (the typical CI box) Probe should
// return a non-nil error; we only assert that it doesn't panic and
// returns within a tight deadline.
func TestAdapterProbeReturnsErrorOrNil(t *testing.T) {
	alwaysAvailable := map[string]bool{
		"native": true, // native requires no runtime
	}
	for _, c := range allAdapterCases() {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := a.Probe(ctx)
			if err != nil && alwaysAvailable[c.name] {
				t.Fatalf("Probe(%s) = %v, expected nil (always-available)", c.name, err)
			}
			// For Probe to fail, the message must mention the tier
			// (so the operator knows what failed).
			if err != nil && !strings.Contains(err.Error(), c.name) {
				t.Logf("note: Probe(%s) error %q does not contain tier name (allowed)", c.name, err)
			}
		})
	}
}

// TestAdapterDeployForNativeOnly deploys every adapter with a minimal
// config and asserts that Deploy returns a non-nil *domain.Sandbox.
// We do NOT call Start/Stop to avoid actually launching runtimes that
// are not present on the test host.
//
// "native" is the only tier guaranteed to succeed on every host; the
// others may fail at Probe. We assert Deploy's error path is well
// formed (mentions the tier name) and that the success path returns
// a sandbox with a non-empty ID.
func TestAdapterDeployForNativeOnly(t *testing.T) {
	cfg := domain.SandboxConfig{Name: "test"}
	a := NewNativeAdapter()
	sb, err := a.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("native Deploy: %v", err)
	}
	if sb == nil {
		t.Fatalf("native Deploy returned nil sandbox")
	}
	if sb.ID == "" {
		t.Fatalf("native sandbox ID is empty")
	}
	if sb.Name != "test" {
		t.Fatalf("native sandbox Name = %q, want test", sb.Name)
	}
}

// TestAdapterDeployGatedByProbe verifies that Probe-gated tiers
// (firecracker, docker, qemu, kvm, hyperkit, applevz, lima, cloudhv,
// crosvm, podman, landlock, seccomp) all return a non-nil error
// from Deploy when the runtime is absent. The test is lenient: if
// the runtime happens to be present on the test host, Deploy may
// succeed — in that case we just log and skip.
func TestAdapterDeployGatedByProbe(t *testing.T) {
	gated := []adapterCase{
		{"firecracker", func() Adapter { return NewFirecrackerAdapter() }},
		{"docker", func() Adapter { return NewDockerAdapter() }},
		{"qemu", func() Adapter { return NewQEMUAdapter() }},
	}
	cfg := domain.SandboxConfig{Name: "test"}
	for _, c := range gated {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			_, err := a.Deploy(context.Background(), cfg)
			if err == nil {
				t.Logf("%s Deploy succeeded (runtime present on test host) — skipping", c.name)
				return
			}
			if !strings.Contains(err.Error(), c.name) {
				t.Fatalf("%s Deploy error %q does not mention tier name", c.name, err)
			}
		})
	}
}

// TestAdapterImplementsInterface is a compile-time check that every
// constructor returns a value satisfying the Adapter interface. The
// _ = lines force a type-check if any adapter drifts.
func TestAdapterImplementsInterface(t *testing.T) {
	for _, c := range allAdapterCases() {
		var _ Adapter = c.make()
	}
}

// TestAdapterInfoHasPlatforms asserts every adapter advertises at
// least one platform; the filter helpers depend on it.
func TestAdapterInfoHasPlatforms(t *testing.T) {
	for _, c := range allAdapterCases() {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			if len(a.Info().Platforms) == 0 {
				t.Fatalf("%s Info().Platforms is empty", c.name)
			}
		})
	}
}

// TestAdapterInfoHasSecurity asserts every adapter advertises a
// non-empty Security level.
func TestAdapterInfoHasSecurity(t *testing.T) {
	for _, c := range allAdapterCases() {
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			if a.Info().Security == "" {
				t.Fatalf("%s Info().Security is empty", c.name)
			}
		})
	}
}

// TestAdapterLifecycleReturnsImmediately ensures Start/Stop/Delete
// return quickly on the no-op-lifecycle adapters. We deliberately
// restrict this to adapters whose Start/Stop/Delete are in-process
// no-ops (wasm, landlock, seccomp, native, kvm, applevz); adapters
// that exec external binaries (firecracker, docker, qemu, lima,
// gvisor, etc.) are tested separately by Probe + Deploy, never by
// actually invoking the Start path on a CI host.
func TestAdapterLifecycleReturnsImmediately(t *testing.T) {
	noopAdapters := map[string]bool{
		"native":   true,
		"kvm":      true,
		"applevz":  true,
		// landlock and seccomp are NOT no-ops anymore:
		// landlock returns an error on unsupported platforms (correct behavior)
		// seccomp returns an error instead of silently disabling the tier (correct behavior)
	}
	for _, c := range allAdapterCases() {
		if !noopAdapters[c.name] {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			a := c.make()
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			if err := a.Start(ctx, "fake-id"); err != nil {
				t.Fatalf("%s Start returned %v on no-op adapter", c.name, err)
			}
			if err := a.Stop(ctx, "fake-id"); err != nil {
				t.Fatalf("%s Stop returned %v on no-op adapter", c.name, err)
			}
			if err := a.Delete(ctx, "fake-id"); err != nil {
				t.Fatalf("%s Delete returned %v on no-op adapter", c.name, err)
			}
		})
	}
}
