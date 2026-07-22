// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

type compositionTestRuntime struct{ sandbox *domain.Sandbox }

func (r compositionTestRuntime) Deploy(context.Context, domain.SandboxConfig) (*domain.Sandbox, error) {
	copy := *r.sandbox
	return &copy, nil
}

type recordingCompositionRuntime struct {
	calls int
	err   error
}

func (r *recordingCompositionRuntime) Deploy(context.Context, domain.SandboxConfig) (*domain.Sandbox, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return &domain.Sandbox{}, nil
}

func TestDeployCompositionValidatesAndAnnotatesHandoff(t *testing.T) {
	sandbox := &domain.Sandbox{Environment: map[string]string{"existing": "value"}}
	engine := NewEngineWithAdapters(compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox}, compositionTestRuntime{sandbox})
	backend, err := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendNanoVMS)
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
	if result.Environment["phenocompose.name"] != "demo" || result.Environment["nvms.backend"] != "nanovms" {
		t.Fatalf("missing handoff labels: %#v", result.Environment)
	}
}

func TestDeployCompositionRejectsMetadataMismatch(t *testing.T) {
	engine := NewEngineWithAdapters(compositionTestRuntime{}, compositionTestRuntime{}, compositionTestRuntime{})
	_, err := engine.DeployComposition(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Backend: nvmsruntime.BackendMetadata{ID: nvmsruntime.BackendPodman, Tier: 2, Lifecycle: true},
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

func TestPodmanNeverInvokesGenericGVisor(t *testing.T) {
	gvisor := &recordingCompositionRuntime{}
	podman := &recordingCompositionRuntime{}
	engine := NewEngineWithAdapters(&recordingCompositionRuntime{}, gvisor, &recordingCompositionRuntime{})
	if err := engine.RegisterBackendDispatcher(nvmsruntime.BackendPodman, podman); err != nil {
		t.Fatal(err)
	}
	backend, err := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendPodman)
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.DeployComposition(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Backend: backend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gvisor.calls != 0 {
		t.Fatalf("Podman invoked generic gVisor %d times", gvisor.calls)
	}
	if podman.calls != 1 {
		t.Fatalf("Podman dispatcher calls = %d, want 1", podman.calls)
	}
}

func TestDeployCompositionWithResourcesRollsBackFailedDeployment(t *testing.T) {
	failing := &recordingCompositionRuntime{err: errors.New("deployment failed")}
	engine := NewEngineWithAdapters(&recordingCompositionRuntime{}, &recordingCompositionRuntime{}, failing)
	store := &gpu.ReservationStore{Path: filepath.Join(t.TempDir(), "reservations.json")}
	if err := engine.ConfigureGPUReservations(store, time.Minute); err != nil {
		t.Fatal(err)
	}
	backend, err := nvmsruntime.NewBackendRegistry().Resolve(nvmsruntime.BackendNanoVMS)
	if err != nil {
		t.Fatal(err)
	}
	manifest := ResourceManifest{
		Version: gpu.ResourceManifestVersion,
		GPUs: []gpu.Device{{
			UUID:         "GPU-11111111-1111-1111-1111-111111111111",
			Name:         "RTX 3090 Ti",
			Architecture: "Ampere",
		}},
	}
	_, err = engine.DeployCompositionWithResources(context.Background(), CompositionRequest{
		Name:    "demo",
		Digest:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Backend: backend,
	}, manifest)
	if err == nil {
		t.Fatal("expected failed deployment")
	}
	active, activeErr := store.Active(context.Background())
	if activeErr != nil {
		t.Fatal(activeErr)
	}
	if len(active) != 0 {
		t.Fatalf("failed deployment leaked reservations: %#v", active)
	}
}
