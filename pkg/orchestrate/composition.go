// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kooshapari/nanovms/internal/domain"
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

	config := request.Config
	config.Name = request.Name
	sandbox, err := e.Deploy(ctx, metadata.Tier, config)
	if err != nil {
		return nil, err
	}
	if sandbox.Environment == nil {
		sandbox.Environment = make(map[string]string)
	}
	sandbox.Environment["phenocompose.name"] = request.Name
	sandbox.Environment["phenocompose.sha256"] = strings.ToLower(request.Digest)
	sandbox.Environment["nvms.backend"] = string(request.Backend.ID)
	return sandbox, nil
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
	if request.Backend.ID == "" || request.Backend.Tier < 1 || !request.Backend.Lifecycle {
		return fmt.Errorf("composition backend metadata is incomplete")
	}
	return nil
}
