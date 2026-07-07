// Package mac provides the macOS adapter with 3-tier VM support:
//   - Tier 1: Native VM (HyperKit, VMware Fusion)
//   - Tier 2: Lima/VZ (container-style virtualization)
//   - Tier 3: MicroVM (Firecracker)
//
// Plus sandbox isolation layers (gVisor, sRAMP, wasmtime).
//
// The adapter is decomposed across this file plus lifecycle.go (Create / Start
// / Stop / Delete / Status / Exec / Pull), lister.go (List + per-runtime
// listers), and sandbox_layer.go (sandbox isolation layers).
package mac

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/kooshapari/nanovms/internal/ports"
)

// Adapter implements RuntimePort for macOS with 3-tier VM support.
type Adapter struct {
	limaPath        string
	hyperkitPath    string
	firecrackerPath string
}

// NewAdapter creates a new macOS adapter with multi-tier support.
func NewAdapter() (*Adapter, error) {
	adapter := &Adapter{}

	// Detect available runtimes
	if limaPath, err := exec.LookPath("limactl"); err == nil {
		adapter.limaPath = limaPath
	} else if colimaPath, err := exec.LookPath("colima"); err == nil {
		adapter.limaPath = colimaPath
	}

	if hkPath, err := exec.LookPath("hyperkit"); err == nil {
		adapter.hyperkitPath = hkPath
	}

	if fcPath, err := exec.LookPath("firecracker"); err == nil {
		adapter.firecrackerPath = fcPath
	}

	if adapter.limaPath == "" && adapter.hyperkitPath == "" && adapter.firecrackerPath == "" {
		return nil, fmt.Errorf("no macOS runtime found: install lima/colima, hyperkit, or firecracker")
	}

	return adapter, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	if a.firecrackerPath != "" {
		return "firecracker"
	}
	if a.hyperkitPath != "" {
		return "hyperkit"
	}
	if a.limaPath != "" {
		if strings.Contains(a.limaPath, "colima") {
			return "colima"
		}
		return "lima"
	}
	return "unknown"
}

// SupportedTiers returns the supported VM tiers.
func (a *Adapter) SupportedTiers() []ports.VMTier {
	var tiers []ports.VMTier
	if a.hyperkitPath != "" {
		tiers = append(tiers, ports.VMTierNative)
	}
	if a.limaPath != "" {
		tiers = append(tiers, ports.VMTierLimaVZ)
	}
	if a.firecrackerPath != "" {
		tiers = append(tiers, ports.VMTierMicroVM)
	}
	return tiers
}
