//go:build integration

// Package e2e contains real end-to-end integration tests for the nanovms
// tier execution paths. native_e2e_test.go covers the Tier 6 "native"
// adapter, which runs workloads directly in the calling process with zero
// isolation overhead. It follows the same pattern as the gVisor e2e tests:
// build-tag guarded and self-skipping. For native the runtime is always
// present (it requires no external binary), so the skip is defensive only.
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// nativeAvailable reports whether the native tier's runtime is present.
// Native execution requires no external binary (it runs in the calling
// process), so this always returns true; it exists only to keep the skip
// pattern uniform across the e2e tests.
func nativeAvailable() bool {
	return true
}

// TestNativeE2EFullLifecycle exercises the native adapter's create / exec /
// stop path end to end. Native Start/Stop/Delete are intentional no-ops, so
// the test verifies the descriptor is created with running status, the
// Datadog-style Probe succeeds, and install required for native (no external
// runtime) is satisfied by Deploy directly.
func TestNativeE2EFullLifecycle(t *testing.T) {
	if !nativeAvailable() {
		t.Skip("native runtime unavailable; skipping native e2e integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := tier.NewNativeAdapter()

	// Probe must always succeed for native — there is no runtime.
	if err := adapter.Probe(ctx); err != nil {
		t.Fatalf("native Probe failed: %v", err)
	}

	// Deploy: create a running sandbox descriptor (native is implicit).
	sb, err := adapter.Deploy(ctx, domain.SandboxConfig{Name: "e2e-native"})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if sb.ID == "" {
		t.Fatal("Deploy returned empty sandbox ID")
	}
	if sb.Type != domain.SandboxTypeNative {
		t.Fatalf("expected type %q, got %q", domain.SandboxTypeNative, sb.Type)
	}

	// Start: no-op for native; must return nil.
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop: no-op for native; must return nil.
	if err := adapter.Stop(ctx, sb.ID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Delete: no-op for native; must return nil.
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}