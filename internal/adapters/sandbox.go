// Package adapters provides the factory that returns the correct SandboxPort
// implementation based on the deployment tier.
package adapters

import (
	"context"
	"fmt"
	"io"

	"github.com/kooshapari/nanovms/internal/adapters/firecracker"
	"github.com/kooshapari/nanovms/internal/adapters/gvisor"
	"github.com/kooshapari/nanovms/internal/adapters/podman"
	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// NewProvider returns an explicitly selected daemon provider. The default
// tier path remains unchanged; provider selection is opt-in at the serve CLI
// boundary so existing tier 1-3 callers retain their behavior.
func NewProvider(name string, tier int) (ports.SandboxPort, error) {
	switch name {
	case "", "tier":
		return NewSandboxPort(tier)
	case "podman":
		return podman.NewAdapter(), nil
	case "apple-containers":
		return &stubAdapter{name: "apple-containers", tier: tier}, nil
	case "wsl-containers":
		return &stubAdapter{name: "wsl-containers", tier: tier}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q (supported: tier, podman, apple-containers, wsl-containers)", name)
	}
}

// stubAdapter is a placeholder for backends that are not yet implemented.
// It satisfies the SandboxPort interface but returns errors on lifecycle operations.
type stubAdapter struct {
	name string
	tier int
}

func (s *stubAdapter) Create(_ context.Context, _ domain.SandboxConfig) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Start(_ context.Context, _ string) error {
	return fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Stop(_ context.Context, _ string, _ bool) error {
	return fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Get(_ context.Context, _ string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) List(_ context.Context) ([]*domain.Sandbox, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Logs(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Exec(_ context.Context, _ string, _ []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}
func (s *stubAdapter) Metrics(_ context.Context, _ string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("%s: adapter not yet implemented", s.name)
}

// NewSandboxPort returns the appropriate SandboxPort implementation for the
// given tier: 1 = WASM (not yet migrated to SandboxPort), 2 = gVisor, 3 = Firecracker.
// Tiers 4-30 return a stub adapter until a real implementation is wired.
func NewSandboxPort(tier int) (ports.SandboxPort, error) {
	switch tier {
	case 1:
		return nil, fmt.Errorf("tier 1 (WASM) SandboxPort adapter not yet implemented")
	case 2:
		return gvisor.NewAdapter(), nil
	case 3:
		return firecracker.NewAdapter(), nil
	default:
		if tier < 4 || tier > 30 {
			return nil, fmt.Errorf("unsupported tier: %d (valid range: 1-30)", tier)
		}
		// Tiers 4-30: return a stub that advertises the tier but fails on lifecycle ops
		// until a real adapter is implemented.
		return &stubAdapter{name: fmt.Sprintf("tier-%d", tier), tier: tier}, nil
	}
}
