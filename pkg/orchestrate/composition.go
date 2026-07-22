// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

// CompositionRequest is the runtime-only handoff from PhenoCompose. The
// digest is the SHA-256 of the rendered composition and is never interpreted
// as cloud or provider state.
type CompositionRequest struct {
	// Name is the stable composition name.
	Name string
	// Digest is a lowercase or uppercase hexadecimal SHA-256 digest.
	Digest string
	// Backend is the selected local execution backend metadata.
	Backend nvmsruntime.BackendMetadata
	// Config is the sandbox configuration to pass to the selected engine tier.
	Config domain.SandboxConfig
}

// DeployComposition validates a PhenoCompose handoff, then invokes the
// existing tier lifecycle and annotates the resulting sandbox with identity
// labels for status correlation.
func (e *Engine) DeployComposition(ctx context.Context, request CompositionRequest) (*domain.Sandbox, error) {
	if e == nil {
		return nil, fmt.Errorf("orchestration engine is nil")
	}
	if err := validateCompositionRequest(request); err != nil {
		return nil, err
	}
	metadata, err := nvmsruntime.NewBackendRegistry().Resolve(request.Backend.ID)
	if err != nil {
		return nil, err
	}
	if metadata.Tier != request.Backend.Tier || metadata.Lifecycle != request.Backend.Lifecycle {
		return nil, fmt.Errorf("backend metadata mismatch for %q", request.Backend.ID)
	}
	if !metadata.Lifecycle {
		return nil, fmt.Errorf("backend %q does not advertise lifecycle support", request.Backend.ID)
	}
	dispatcher, exists := e.backendDispatchers[request.Backend.ID]
	if !exists || dispatcher == nil {
		return nil, fmt.Errorf("backend %q has no backend-specific dispatcher", request.Backend.ID)
	}

	config := request.Config
	config.Name = request.Name
	sandbox, err := dispatcher.Deploy(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("backend %q deployment failed: %w", request.Backend.ID, err)
	}
	if sandbox == nil {
		return nil, fmt.Errorf("backend %q dispatcher returned a nil sandbox", request.Backend.ID)
	}
	if sandbox.Environment == nil {
		sandbox.Environment = make(map[string]string)
	}
	sandbox.Environment["phenocompose.name"] = request.Name
	sandbox.Environment["phenocompose.sha256"] = strings.ToLower(request.Digest)
	sandbox.Environment["nvms.backend"] = string(request.Backend.ID)
	return sandbox, nil
}

// ResourceManifest is the versioned companion to CompositionRequest.
type ResourceManifest = gpu.ResourceManifest

// DeployCompositionWithResources validates deterministic GPU evidence,
// reserves every requested UUID atomically, and rolls the reservation back if
// backend deployment fails.
func (e *Engine) DeployCompositionWithResources(ctx context.Context, request CompositionRequest, manifest ResourceManifest) (*domain.Sandbox, error) {
	canonical, err := manifest.CanonicalJSON()
	if err != nil {
		return nil, fmt.Errorf("invalid resource manifest: %w", err)
	}
	if len(manifest.GPUs) == 0 {
		return e.DeployComposition(ctx, request)
	}
	if e == nil || e.reservations == nil {
		return nil, fmt.Errorf("GPU reservation store is not configured")
	}
	uuids := make([]gpu.UUID, len(manifest.GPUs))
	for i := range manifest.GPUs {
		uuids[i] = manifest.GPUs[i].UUID
	}
	owner := request.Name + ":" + strings.ToLower(request.Digest)
	lease, err := e.reservations.Reserve(ctx, uuids, owner, e.reservationTTL)
	if err != nil {
		return nil, fmt.Errorf("reserve composition GPUs: %w", err)
	}
	sandbox, deployErr := e.DeployComposition(ctx, request)
	if deployErr != nil {
		if rollbackErr := e.reservations.Release(ctx, lease); rollbackErr != nil {
			return nil, fmt.Errorf("%w (GPU reservation rollback failed: %v)", deployErr, rollbackErr)
		}
		return nil, deployErr
	}
	digest := sha256.Sum256(canonical)
	if sandbox.Environment == nil {
		sandbox.Environment = make(map[string]string)
	}
	sandbox.Environment["nvms.resources.version"] = manifest.Version
	sandbox.Environment["nvms.resources.sha256"] = hex.EncodeToString(digest[:])
	sandbox.Environment["nvms.gpu.reservation.owner"] = lease.Owner
	sandbox.Environment["nvms.gpu.reservation.token"] = lease.Token
	sandbox.Environment["nvms.gpu.reservation.expires"] = lease.ExpiresAt.Format(time.RFC3339Nano)
	return sandbox, nil
}

// ReleaseCompositionResources releases a successful deployment's GPU lease.
func (e *Engine) ReleaseCompositionResources(ctx context.Context, sandbox *domain.Sandbox, manifest ResourceManifest) error {
	if e == nil || e.reservations == nil || sandbox == nil {
		return fmt.Errorf("engine, reservation store, and sandbox are required")
	}
	environment := sandbox.Environment
	expiresAt, err := time.Parse(time.RFC3339Nano, environment["nvms.gpu.reservation.expires"])
	if err != nil {
		return fmt.Errorf("invalid reservation expiry metadata: %w", err)
	}
	uuids := make([]gpu.UUID, len(manifest.GPUs))
	for i := range manifest.GPUs {
		uuids[i] = manifest.GPUs[i].UUID
	}
	return e.reservations.Release(ctx, gpu.ReservationLease{
		Token:     environment["nvms.gpu.reservation.token"],
		Owner:     environment["nvms.gpu.reservation.owner"],
		UUIDs:     uuids,
		ExpiresAt: expiresAt,
	})
}

func validateCompositionRequest(request CompositionRequest) error {
	if request.Name == "" || strings.TrimSpace(request.Name) != request.Name || strings.ContainsAny(request.Name, " \t\r\n") {
		return fmt.Errorf("composition name must be non-empty and contain no whitespace")
	}
	if len(request.Digest) != 64 {
		return fmt.Errorf("composition digest must be a 64-character SHA-256 hex string")
	}
	if _, err := hex.DecodeString(request.Digest); err != nil {
		return fmt.Errorf("composition digest must be hexadecimal: %w", err)
	}
	if request.Backend.ID == "" || request.Backend.Tier < 1 {
		return fmt.Errorf("composition backend metadata is incomplete")
	}
	return nil
}
