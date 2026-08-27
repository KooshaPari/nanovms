package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/kooshapari/nanovms/internal/adapters"
	"github.com/kooshapari/nanovms/internal/config"
	"github.com/kooshapari/nanovms/internal/domain"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "stop":
		stopCmd(os.Args[2:])
	case "list":
		listCmd()
	case "tier":
		tierCmd(os.Args[2:])
	case "version":
		fmt.Println("nanovms 0.1.0")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "nanovms: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: nanovms <command> [flags]

Commands:
  run <config.toml>         Deploy a workload from a TOML config file
  stop <sandbox-id>         Stop and remove a sandbox
  list                      List running sandboxes
  tier list                 List available sandbox tiers
  tier probe <tier>         Probe a tier for available capabilities
  version                   Print version
  help                      Show this help

Examples:
  nanovms run my-sandbox.toml
  nanovms stop abc123
  nanovms list
  nanovms tier list
  nanovms tier probe 3`)
}

// runCmd deploys a workload from a TOML configuration file.
func runCmd(args []string) {
	runSet := flag.NewFlagSet("run", flag.ExitOnError)
	platform := runSet.String("platform", "", "Override platform (linux|mac|windows)")
	tierOverride := runSet.Int("tier", 0, "Override tier from config")
	_ = runSet.Parse(args)

	if runSet.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: nanovms run [--platform <p>] [--tier <n>] <config.toml>")
		os.Exit(1)
	}

	cfgPath := runSet.Arg(0)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	tier := cfg.Tier
	if *tierOverride > 0 {
		tier = *tierOverride
	}

	plat := *platform
	if plat == "" {
		plat = detectPlatform()
	}

	fmt.Printf("Deploying %q on tier %d (platform=%s)\n", cfg.Name, tier, plat)

	ctx := context.Background()

	adapter, err := adapters.NewProvider("tier", tier)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating adapter: %v\n", err)
		os.Exit(1)
	}

	sandboxCfg := domain.SandboxConfig{
		VMType:      domain.VMFlavor(fmt.Sprintf("tier-%d", tier)),
		SandboxType: domain.SandboxTypeContainer,
		Labels: map[string]string{
			"nanovms.name":  cfg.Name,
			"nanovms.image": cfg.Image,
			"nanovms.tier":  fmt.Sprintf("%d", tier),
		},
	}

	sb, err := adapter.Create(ctx, sandboxCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating sandbox: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sandbox created: %s (name=%s, tier=%d)\n", sb.ID, sb.Name, tier)
	fmt.Printf("Status: %s\n", sb.Status)

	if err := adapter.Start(ctx, sb.ID); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not start sandbox: %v\n", err)
	}

	fmt.Println("Done.")
}

// stopCmd stops and removes a sandbox by ID.
func stopCmd(args []string) {
	stopSet := flag.NewFlagSet("stop", flag.ExitOnError)
	force := stopSet.Bool("force", false, "Force stop without graceful shutdown")
	_ = stopSet.Parse(args)

	if stopSet.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: nanovms stop [--force] <sandbox-id>")
		os.Exit(1)
	}

	id := stopSet.Arg(0)
	ctx := context.Background()

	adapter, err := adapters.NewProvider("tier", 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating adapter: %v\n", err)
		os.Exit(1)
	}

	if err := adapter.Stop(ctx, id, *force); err != nil {
		fmt.Fprintf(os.Stderr, "error stopping sandbox %s: %v\n", id, err)
		os.Exit(1)
	}
	if err := adapter.Delete(ctx, id); err != nil {
		fmt.Fprintf(os.Stderr, "error removing sandbox %s: %v\n", id, err)
		os.Exit(1)
	}

	fmt.Printf("Sandbox %s stopped.\n", id)
}

// listCmd lists running sandboxes.
func listCmd() {
	fmt.Println("ID\t\tNAME\tSTATUS\tTIER")
	fmt.Println("--\t\t----\t------\t----")
	fmt.Println("(connect to nvms daemon with: nvms vm list)")
}

// tierCmd manages sandbox tiers.
func tierCmd(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Println(`Usage: nanovms tier <subcommand>

Subcommands:
  list                 List all known tiers
  probe <tier-number>  Probe a tier for capabilities`)
		return
	}

	switch args[0] {
	case "list":
		tierListCmd()
	case "probe":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: nanovms tier probe <tier-number>")
			os.Exit(1)
		}
		tierProbeCmd(args[1])
	default:
		fmt.Fprintf(os.Stderr, "nanovms tier: unknown subcommand %q\n", args[0])
		os.Exit(1)
	}
}

func tierListCmd() {
	tiers := []struct {
		num  int
		name string
		desc string
	}{
		{1, "WASM", "WebAssembly sandbox (<1ms startup)"},
		{2, "gVisor", "User-space kernel sandbox (~100ms)"},
		{3, "Firecracker", "MicroVM (~125ms startup)"},
		{4, "QEMU", "Full virtual machine (~2s)"},
		{5, "Podman", "OCI container runtime"},
		{6, "Landlock", "Linux kernel sandboxing"},
		{7, "Seccomp", "System call filtering"},
		{8, "Native", "bwrap/firejail sandboxing"},
		{9, "LXC", "Linux Containers"},
		{10, "Docker", "Docker runtime"},
	}

	fmt.Println("TIER\tNAME\t\tDESCRIPTION")
	fmt.Println("----\t----\t\t-----------")
	for _, t := range tiers {
		name := t.name
		if len(name) < 8 {
			name += "\t"
		}
		fmt.Printf("%d\t%s\t%s\n", t.num, name, t.desc)
	}
}

func tierProbeCmd(tierStr string) {
	tier := 0
	if _, err := fmt.Sscanf(tierStr, "%d", &tier); err != nil || tier < 1 || tier > 30 {
		fmt.Fprintf(os.Stderr, "invalid tier number: %s (must be 1-30)\n", tierStr)
		os.Exit(1)
	}

	if _, err := adapters.NewProvider("tier", tier); err != nil {
		fmt.Printf("Tier %d: NOT AVAILABLE (%v)\n", tier, err)
		return
	}

	fmt.Printf("Tier %d: AVAILABLE\n", tier)
}

func detectPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// isLinux checks if the current platform is Linux.
func isLinux() bool {
	return strings.Contains(detectPlatform(), "linux")
}
