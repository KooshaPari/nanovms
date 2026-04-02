package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/kooshapari/nanovms/internal/adapters/linux"
	"github.com/kooshapari/nanovms/internal/adapters/mac"
	"github.com/kooshapari/nanovms/internal/adapters/windows"
	"github.com/kooshapari/nanovms/internal/domain"
)

// Deunan provides multi-platform VM and sandbox orchestration.
type Deunan struct {
	vmAdapters map[string]domain.VMAdapter
}

var (
	platform   string
	vmTier     string
	sandboxOpt string
	name       string
	image      string
)

func init() {
	flag.StringVar(&platform, "platform", "", "Target platform (mac|windows|linux|auto)")
	flag.StringVar(&vmTier, "vm-tier", "auto", "VM isolation tier (native|lima|microvm|auto)")
	flag.StringVar(&sandboxOpt, "sandbox", "auto", "Sandbox isolation (gvisor|landlock|seccomp|windows|none|auto)")
	flag.StringVar(&name, "name", "", "Sandbox name")
	flag.StringVar(&image, "image", "", "OCI image to use")
}

func main() {
	flag.Parse()

	ctx := context.Background()

	// Auto-detect platform if not specified
	if platform == "" {
		switch runtime.GOOS {
		case "darwin":
			platform = "mac"
		case "windows":
			platform = "windows"
		default:
			platform = "linux"
		}
	}

	// Determine VM tier
	vmTier = resolveVMTier(platform, vmTier)

	// Determine sandbox tier
	sandboxOpt = resolveSandboxTier(platform, sandboxOpt)

	fmt.Printf("Platform: %s | VM Tier: %s | Sandbox: %s\n", platform, vmTier, sandboxOpt)

	// Create the appropriate VM adapter
	vmAdapter, err := createVMAdapter(platform, vmTier)
	if err != nil {
		log.Fatal(err)
	}

	// List sandboxes
	sandboxes, err := vmAdapter.List(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sandboxes (%d):\n", len(sandboxes))
	for _, sb := range sandboxes {
		fmt.Printf("  - %s (status: %s, vm: %s, sandbox: %s)\n",
			sb.ID, sb.Status, sb.VMTier, sb.SandboxTier)
	}

	// Create a test sandbox if name provided
	if name != "" {
		id, err := vmAdapter.Create(ctx, &domain.SandboxConfig{
			Name:         name,
			Image:        image,
			VMTier:      vmTier,
			SandboxTier: sandboxOpt,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created sandbox: %s\n", id)
	}
}

func resolveVMTier(platform, tier string) string {
	if tier == "auto" {
		switch platform {
		case "mac":
			if mac.IsLimaAvailable() {
				return "lima"
			}
			return "native"
		case "windows":
			if windows.IsWSLAvailable() {
				return "wsl"
			}
			return "hyperv"
		case "linux":
			return "native"
		}
	}
	return tier
}

func resolveSandboxTier(platform, tier string) string {
	if tier == "auto" {
		switch platform {
		case "mac":
			return "none"
		case "windows":
			if windows.IsWindowsSandboxAvailable() {
				return "windows"
			}
			return "none"
		case "linux":
			if linux.IsGVisorAvailable() {
				return "gvisor"
			}
			if linux.IsLandlockAvailable() {
				return "landlock"
			}
			return "seccomp"
		}
	}
	return tier
}

func createVMAdapter(platform, tier string) (domain.VMAdapter, error) {
	switch platform {
	case "mac":
		return mac.NewVMAdapter(tier), nil
	case "windows":
		return windows.NewVMAdapter(tier)
	case "linux":
		return linux.NewVMAdapter(tier)
	default:
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
}
