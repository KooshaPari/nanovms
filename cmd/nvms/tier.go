// SPDX-License-Identifier: MIT OR Apache-2.0
// Package main — tier.go implements the `nvms tier` subcommand: list,
// inspect, and probe sandbox tiers. After tier expansion (15 -> 30)
// this command surfaces every adapter registered in the default
// registry.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/kooshapari/nanovms/pkg/tier"
)

// tierCmd dispatches the `nvms tier` subcommand. Supported subcommands:
//
//	nvms tier list                       Print all registered tiers.
//	nvms tier list --json                Print all tiers as JSON.
//	nvms tier list --security <lvl>      Filter by security level (low/medium/high/untrusted).
//	nvms tier info <name>                Print details for a single tier.
//	nvms tier probe <name>               Probe a single tier's runtime availability.
func tierCmd(args []string) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`Usage: nvms tier <subcommand> [args]

Subcommands:
  list [--security <low|medium|high|untrusted>] [--json]
                                  List registered sandbox tiers.
  info <name>                     Show details for one tier.
  probe <name>                    Probe a tier's runtime availability.

Examples:
  nvms tier list
  nvms tier list --json
  nvms tier list --security high
  nvms tier info firecracker
  nvms tier probe kata`)
		return
	}

	switch args[0] {
	case "list":
		tierListCmd(args[1:])
	case "info":
		tierInfoCmd(args[1:])
	case "probe":
		tierProbeCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "nvms tier: unknown subcommand %q\n", args[0])
		os.Exit(1)
	}
}

// tierListCmd implements `nvms tier list`.
func tierListCmd(args []string) {
	asJSON := false
	security := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json", "-j":
			asJSON = true
		case "--security", "-s":
			if i+1 >= len(args) {
				log.Fatal("--security requires an argument (low|medium|high|untrusted)")
			}
			security = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "nvms tier list: unknown flag %q\n", args[i])
			os.Exit(1)
		}
	}

	r := tier.DefaultRegistry()
	all := r.Info()

	// Filter by security level if requested, then sort for stable output.
	names := make([]string, 0, len(all))
	for n, info := range all {
		if security != "" && info.Security != security {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)

	if asJSON {
		type jsonEntry struct {
			Name        string   `json:"name"`
			DisplayName string   `json:"display_name"`
			Description string   `json:"description"`
			StartupMS   int      `json:"startup_ms"`
			MemoryMB    int      `json:"memory_mb"`
			Security    string   `json:"security"`
			Platforms   []string `json:"platforms"`
			Workloads   []string `json:"workloads"`
		}
		entries := make([]jsonEntry, 0, len(names))
		for _, n := range names {
			info := all[n]
			entries = append(entries, jsonEntry{
				Name:        info.Name,
				DisplayName: info.DisplayName,
				Description: info.Description,
				StartupMS:   info.StartupMS,
				MemoryMB:    info.MemoryMB,
				Security:    info.Security,
				Platforms:   info.Platforms,
				Workloads:   info.Workloads,
			})
		}
		out, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			log.Fatalf("json marshal: %v", err)
		}
		fmt.Println(string(out))
		return
	}

	// Human-readable table.
	fmt.Printf("Registered tiers: %d\n\n", len(names))
	fmt.Printf("%-18s %-9s %-10s %-10s %-30s\n", "NAME", "STARTUP", "MEMORY", "SECURITY", "PLATFORMS")
	fmt.Printf("%-18s %-9s %-10s %-10s %-30s\n", "----", "-------", "------", "--------", "---------")
	for _, n := range names {
		info := all[n]
		platforms := joinShort(info.Platforms)
		fmt.Printf("%-18s %-9d %-10d %-10s %-30s\n",
			info.Name, info.StartupMS, info.MemoryMB, info.Security, platforms)
	}
}

// tierInfoCmd implements `nvms tier info <name>`.
func tierInfoCmd(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: nvms tier info <name>")
	}
	name := args[0]
	r := tier.DefaultRegistry()
	all := r.Info()
	i, ok := all[name]
	if !ok {
		log.Fatalf("nvms tier info: %q not registered (have: %v)", name, r.Names())
	}
	fmt.Printf("Name:        %s\n", i.Name)
	if i.DisplayName != "" {
		fmt.Printf("Display:     %s\n", i.DisplayName)
	}
	if i.Description != "" {
		fmt.Printf("Description: %s\n", i.Description)
	}
	fmt.Printf("Startup:     %dms\n", i.StartupMS)
	fmt.Printf("Memory:      %dMB\n", i.MemoryMB)
	fmt.Printf("Security:    %s\n", i.Security)
	if len(i.Platforms) > 0 {
		fmt.Printf("Platforms:   %v\n", i.Platforms)
	}
	if len(i.Workloads) > 0 {
		fmt.Printf("Workloads:   %v\n", i.Workloads)
	}
}

// tierProbeCmd implements `nvms tier probe <name>`.
func tierProbeCmd(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: nvms tier probe <name>")
	}
	name := args[0]
	r := tier.DefaultRegistry()
	a, err := r.Get(name)
	if err != nil {
		log.Fatalf("nvms tier probe: %v", err)
	}
	if err := a.Probe(context.Background()); err != nil {
		fmt.Printf("probe %s: FAIL: %v\n", name, err)
		os.Exit(1)
	}
	fmt.Printf("probe %s: OK\n", name)
}

// joinShort formats a platforms list for table display.
func joinShort(p []string) string {
	if len(p) == 0 {
		return "(any)"
	}
	out := ""
	for i, s := range p {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
