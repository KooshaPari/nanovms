// Package adapters provides the factory that returns the correct SandboxPort
// implementation based on the deployment tier.
package adapters

import (
	"fmt"

	"github.com/kooshapari/nanovms/internal/adapters/containers"
	"github.com/kooshapari/nanovms/internal/adapters/firecracker"
	"github.com/kooshapari/nanovms/internal/adapters/gvisor"
	"github.com/kooshapari/nanovms/internal/adapters/podman"
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
	case "apple-containers", "apple":
		return containers.NewAppleAdapter(""), nil
	case "wsl-containers", "wsl":
		return containers.NewWSLAdapter(""), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q (supported: tier, podman, apple-containers, wsl-containers)", name)
	}
}

// NewSandboxPort returns the appropriate SandboxPort implementation for the
// given tier: 1 = WASM (not yet migrated to SandboxPort), 2 = gVisor, 3 = Firecracker.
// For tier 1, it returns an error until a WASM SandboxPort adapter is created.
func NewSandboxPort(tier int) (ports.SandboxPort, error) {
	switch tier {
	case 1:
		return nil, fmt.Errorf("tier 1 (WASM) SandboxPort adapter not yet implemented")
	case 2:
		return gvisor.NewAdapter(), nil
	case 3:
		return firecracker.NewAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported tier: %d", tier)
	}
}
