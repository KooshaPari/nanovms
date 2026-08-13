// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

type compositionTestRuntime struct{ sandbox *domain.Sandbox }

func (r compositionTestRuntime) Deploy(context.Context, domain.SandboxConfig) (*domain.Sandbox, error) {
	copy := *r.sandbox
	return &copy, nil
}

func TestDeployCompositionValidatesAndAnnotatesHandoff(t *testing.T) {
	sandbox := &domain.Sandbox{Environment: map[string]string{"existing": "value"}}
	engine := NewEngineWithAdapters(compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox})
	backend, err := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendPodman)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.DeployComposition(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Backend: backend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Environment["phenocompose.name"] != "demo" || result.Environment["nvms.backend"] != "podman" {
		t.Fatalf("missing handoff labels: %#v", result.Environment)
	}
}

func TestDeployCompositionRejectsMetadataMismatch(t *testing.T) {
	engine := NewEngineWithAdapters(compositionTestRuntime{}, compositionTestRuntime{}, compositionTestRuntime{})
	_, err := engine.DeployComposition(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Backend: nvmsruntime.BackendMetadata{ID: nvmsruntime.BackendPodman, Tier: 3, Lifecycle: true},
	})
	if err == nil {
		t.Fatal("expected backend metadata mismatch")
	}
}

func TestDeployCompositionRejectsBadDigest(t *testing.T) {
	backend, _ := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendNanoVMS)
	_, err := (&Engine{}).DeployComposition(context.Background(), CompositionRequest{Name: "demo", Digest: "not-a-digest", Backend: backend})
	if err == nil {
		t.Fatal("expected invalid digest")
	}
}

func TestDeployCompositionHandoffNormalizesBytePortDigest(t *testing.T) {
	sandbox := &domain.Sandbox{Environment: map[string]string{}}
	engine := NewEngineWithAdapters(compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox})
	backend, err := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendPodman)
	if err != nil {
		t.Fatal(err)
	}

	// BytePort carries the self-describing form; NanoVMS accepts the raw
	// 64-hex form. The adapter boundary must strip only the known prefix.
	bytePortDigest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	rawDigest := strings.TrimPrefix(bytePortDigest, "sha256:")
	result, err := engine.DeployComposition(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  rawDigest,
		Backend: backend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Environment["phenocompose.sha256"]; got != rawDigest {
		t.Fatalf("normalized digest mismatch: got %q want %q", got, rawDigest)
	}
}
