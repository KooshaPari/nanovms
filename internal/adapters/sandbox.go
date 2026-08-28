// Package adapters provides the factory that returns the correct SandboxPort
// implementation based on the deployment tier.
package adapters

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/kooshapari/nanovms/internal/adapters/firecracker"
	"github.com/kooshapari/nanovms/internal/adapters/gvisor"
	"github.com/kooshapari/nanovms/internal/adapters/podman"
	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
	"github.com/kooshapari/nanovms/pkg/tier"
)

// tierAdapter wraps a tier.Adapter (which has Deploy/Start/Stop/Delete) to
// implement the ports.SandboxPort interface. This bridges the two different
// abstraction layers.
type tierAdapter struct {
	inner     tier.Adapter
	tierID    int
	sandboxes map[string]*domain.Sandbox // track created sandboxes
	mu        sync.RWMutex
}

func (t *tierAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	sandbox, err := t.inner.Deploy(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("tier %d deploy: %w", t.tierID, err)
	}
	if sandbox == nil {
		sandbox = &domain.Sandbox{
			ID:     fmt.Sprintf("sandbox-%s-%d", config.Name, t.tierID),
			Name:   fmt.Sprintf("tier-%d", t.tierID),
			Status: domain.SandboxStatusPending,
			Config: &config,
		}
	}
	t.mu.Lock()
	t.sandboxes[sandbox.ID] = sandbox
	t.mu.Unlock()
	return sandbox, nil
}

func (t *tierAdapter) Start(ctx context.Context, id string) error {
	return t.inner.Start(ctx, id)
}

func (t *tierAdapter) Stop(ctx context.Context, id string, _ bool) error {
	return t.inner.Stop(ctx, id)
}

func (t *tierAdapter) Delete(ctx context.Context, id string) error {
	err := t.inner.Delete(ctx, id)
	t.mu.Lock()
	delete(t.sandboxes, id)
	t.mu.Unlock()
	return err
}

func (t *tierAdapter) Get(_ context.Context, id string) (*domain.Sandbox, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if s, ok := t.sandboxes[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("sandbox %s not found in tier %d", id, t.tierID)
}

func (t *tierAdapter) List(_ context.Context) ([]*domain.Sandbox, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*domain.Sandbox, 0, len(t.sandboxes))
	for _, s := range t.sandboxes {
		result = append(result, s)
	}
	return result, nil
}

func (t *tierAdapter) Logs(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("tier %d: logs not supported via tier adapter", t.tierID)
}

func (t *tierAdapter) Exec(_ context.Context, _ string, _ []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("tier %d: exec not supported via tier adapter", t.tierID)
}

func (t *tierAdapter) Metrics(_ context.Context, _ string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("tier %d: metrics not supported via tier adapter", t.tierID)
}

// NewProvider returns an explicitly selected daemon provider. The default
// tier path remains unchanged; provider selection is opt-in at the serve CLI
// boundary so existing tier 1-3 callers retain their behavior.
func NewProvider(name string, tierID int) (ports.SandboxPort, error) {
	switch name {
	case "", "tier":
		return NewSandboxPort(tierID)
	case "podman":
		return podman.NewAdapter(), nil
	case "apple-containers":
		return &stubAdapter{name: "apple-containers", tier: tierID}, nil
	case "wsl-containers":
		return &stubAdapter{name: "wsl-containers", tier: tierID}, nil
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

// tierName maps tier IDs to canonical adapter names for registry lookup.
var tierName = map[int]string{
	4: "docker", 5: "podman", 6: "gvisor", 7: "firecracker",
	8: "native", 9: "landlock", 10: "seccomp",
	11: "lxc", 12: "kata", 13: "qemu", 14: "kvm",
	15: "wasm", 16: "wasmtime", 17: "cloudhv", 18: "crosvm",
	19: "applevz", 20: "hyperkit", 21: "lima",
	22: "firejail", 23: "bubblewrap", 24: "nsjail",
	25: "youki", 26: "crun", 27: "sysbox", 28: "kata-fc",
	29: "kata-qemu", 30: "kata-cloud-hv",
}

// NewSandboxPort returns the appropriate SandboxPort implementation for the
// given tier. Tiers 1-3 use existing adapters. Tiers 4-30 use the tier
// registry to find and wrap real adapter implementations.
func NewSandboxPort(tierID int) (ports.SandboxPort, error) {
	switch tierID {
	case 1:
		return nil, fmt.Errorf("tier 1 (WASM) SandboxPort adapter not yet implemented")
	case 2:
		return gvisor.NewAdapter(), nil
	case 3:
		return firecracker.NewAdapter(), nil
	default:
		if tierID < 4 || tierID > 30 {
			return nil, fmt.Errorf("unsupported tier: %d (valid range: 1-30)", tierID)
		}
		// Look up the adapter name for this tier
		name, ok := tierName[tierID]
		if !ok {
			return nil, fmt.Errorf("no adapter mapping for tier %d", tierID)
		}

		// Try to get the adapter from the tier registry
		registry := tier.DefaultRegistry()
		adapter, err := registry.Get(name)
		if err != nil {
			// Adapter exists in registry but name doesn't match — try case-insensitive
			for _, regName := range registry.List() {
				if strings.EqualFold(regName, name) {
					adapter, err = registry.Get(regName)
					if err == nil {
						break
					}
				}
			}
		}

		if err != nil || adapter == nil {
			// Tier not in registry — return stub
			return &stubAdapter{name: fmt.Sprintf("tier-%d", tierID), tier: tierID}, nil
		}

		return &tierAdapter{
			inner:     adapter,
			tierID:    tierID,
			sandboxes: make(map[string]*domain.Sandbox),
		}, nil
	}
}

