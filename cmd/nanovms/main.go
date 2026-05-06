package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"

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

	// Create a test sandbox if name provided
	if name != "" {
		sandbox, err := createSandbox(ctx, platform, vmTier, sandboxOpt, name, image)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created sandbox: %s\n", sandbox.ID)
	}
}

func resolveVMTier(platform, tier string) string {
	if tier == "auto" {
		switch platform {
		case "mac":
			return "lima"
		case "windows":
			return "wsl"
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
			return "none"
		case "linux":
			return "seccomp"
		}
	}
	return tier
}

func createSandbox(ctx context.Context, platform, vmTier, sandboxOpt, name, image string) (*domain.Sandbox, error) {
	config := domain.SandboxConfig{
		Name: name,
		Image: image,
		VMType: domain.VMFlavor(vmTier),
		SandboxType: domain.SandboxType(sandboxOpt),
	}

	// Create sandbox using the adapter
	sandbox := &domain.Sandbox{
		ID:     domain.GenerateID(),
		Name:   name,
		Status: domain.SandboxStatusPending,
		Config: &config,
	}

	return sandbox, nil
}
