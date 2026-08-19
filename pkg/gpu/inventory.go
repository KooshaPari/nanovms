// SPDX-License-Identifier: MIT OR Apache-2.0
package gpu

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var inventoryQueryArgs = []string{
	"--query-gpu=uuid,index,name,compute_cap,driver_version",
	"--format=csv,noheader,nounits",
}

// InventoryAdapter obtains one scoped NVIDIA inventory.
type InventoryAdapter interface {
	Inventory(ctx context.Context) (ScopedInventory, error)
}

// InventoryProvider owns authoritative NanoVMS hardware discovery.
type InventoryProvider interface {
	Inventory(ctx context.Context) ([]Device, error)
}

// ReconciledInventoryProvider queries configured scopes and reconciles devices
// strictly by canonical UUID.
type ReconciledInventoryProvider struct {
	Adapters []InventoryAdapter
}

// Inventory returns one authoritative, UUID-reconciled hardware inventory.
func (provider ReconciledInventoryProvider) Inventory(ctx context.Context) ([]Device, error) {
	devices, _, err := provider.InventoryWithScopes(ctx)
	return devices, err
}

// InventoryWithScopes returns reconciled devices and every adapter scope in one pass.
func (provider ReconciledInventoryProvider) InventoryWithScopes(ctx context.Context) ([]Device, []ScopedInventory, error) {
	if len(provider.Adapters) == 0 {
		return nil, nil, fmt.Errorf("at least one inventory adapter is required")
	}
	inventories := make([]ScopedInventory, 0, len(provider.Adapters))
	for _, adapter := range provider.Adapters {
		if adapter == nil {
			return nil, nil, fmt.Errorf("inventory adapter is required")
		}
		inventory, err := adapter.Inventory(ctx)
		if err != nil {
			return nil, nil, err
		}
		inventories = append(inventories, inventory)
	}
	devices, err := Reconcile(inventories...)
	if err != nil {
		return nil, inventories, err
	}
	return devices, inventories, nil
}

// ExecutionScopedInventory returns the scope used for evaluation execution probes.
func ExecutionScopedInventory(inventories []ScopedInventory, distribution string) (*ScopedInventory, error) {
	if distribution == "" {
		for i := range inventories {
			if inventories[i].Scope == ScopeWindowsHost {
				inventory := inventories[i]
				return &inventory, nil
			}
		}
		return nil, fmt.Errorf("windows host inventory is missing")
	}
	for i := range inventories {
		if inventories[i].Scope == ScopeWSLDistro && inventories[i].ScopeID == distribution {
			inventory := inventories[i]
			return &inventory, nil
		}
	}
	return nil, fmt.Errorf("WSL distribution %q inventory is missing", distribution)
}

// WindowsInventoryAdapter queries the Windows host's native nvidia-smi.
type WindowsInventoryAdapter struct {
	Runner CommandRunner
}

func (adapter WindowsInventoryAdapter) Inventory(ctx context.Context) (ScopedInventory, error) {
	return runInventory(ctx, adapter.Runner, ScopeWindowsHost, "", "nvidia-smi", nil)
}

// WSLInventoryAdapter queries one explicitly named WSL distribution.
type WSLInventoryAdapter struct {
	Runner       CommandRunner
	Distribution string
}

func (adapter WSLInventoryAdapter) Inventory(ctx context.Context) (ScopedInventory, error) {
	if strings.TrimSpace(adapter.Distribution) == "" {
		return ScopedInventory{}, fmt.Errorf("WSL distribution is required")
	}
	prefix := []string{"-d", adapter.Distribution, "--", "nvidia-smi"}
	return runInventory(ctx, adapter.Runner, ScopeWSLDistro, adapter.Distribution, "wsl.exe", prefix)
}

// RuntimeInventoryAdapter queries an optional runtime-specific nvidia-smi.
// Prefix contains arguments before nvidia-smi query arguments.
type RuntimeInventoryAdapter struct {
	Runner  CommandRunner
	Runtime string
	Command string
	Prefix  []string
}

func (adapter RuntimeInventoryAdapter) Inventory(ctx context.Context) (ScopedInventory, error) {
	if strings.TrimSpace(adapter.Runtime) == "" || strings.TrimSpace(adapter.Command) == "" {
		return ScopedInventory{}, fmt.Errorf("runtime name and command are required")
	}
	return runInventory(ctx, adapter.Runner, ScopeRuntime, adapter.Runtime, adapter.Command, adapter.Prefix)
}

func runInventory(ctx context.Context, runner CommandRunner, scope InventoryScope, scopeID, command string, prefix []string) (ScopedInventory, error) {
	if runner == nil {
		return ScopedInventory{}, fmt.Errorf("command runner is required")
	}
	queryArgs := append(append([]string(nil), prefix...), inventoryQueryArgs...)
	query, err := runner.Run(ctx, command, queryArgs...)
	if err != nil {
		return ScopedInventory{}, fmt.Errorf("%s GPU inventory failed: %w", scope, err)
	}
	xmlArgs := append(append([]string(nil), prefix...), "-q", "-x")
	details, err := runner.Run(ctx, command, xmlArgs...)
	if err != nil {
		return ScopedInventory{}, fmt.Errorf("%s GPU driver capability query failed: %w", scope, err)
	}
	ceiling, err := parseDriverCUDACeiling(details.Stdout)
	if err != nil {
		return ScopedInventory{}, fmt.Errorf("%s GPU driver capability output: %w", scope, err)
	}
	return parseInventoryCSV(query.Stdout, scope, scopeID, ceiling)
}

func parseInventoryCSV(output []byte, scope InventoryScope, scopeID, ceiling string) (ScopedInventory, error) {
	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.TrimLeadingSpace = true
	inventory := ScopedInventory{Scope: scope, ScopeID: scopeID, DriverCUDACeiling: ceiling}
	seen := make(map[UUID]struct{})
	for rowNumber := 1; ; rowNumber++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ScopedInventory{}, fmt.Errorf("malformed nvidia-smi CSV row %d: %w", rowNumber, err)
		}
		if len(row) != 5 {
			return ScopedInventory{}, fmt.Errorf("nvidia-smi CSV row %d has %d fields, want 5", rowNumber, len(row))
		}
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		uuid, err := ParseUUID(row[0])
		if err != nil {
			return ScopedInventory{}, fmt.Errorf("nvidia-smi CSV row %d: %w", rowNumber, err)
		}
		if _, duplicate := seen[uuid]; duplicate {
			return ScopedInventory{}, fmt.Errorf("duplicate GPU UUID %q in %s inventory", uuid, scope)
		}
		seen[uuid] = struct{}{}
		index, err := strconv.Atoi(row[1])
		if err != nil || index < 0 {
			return ScopedInventory{}, fmt.Errorf("nvidia-smi CSV row %d has invalid index %q", rowNumber, row[1])
		}
		if row[2] == "" || row[3] == "" || row[4] == "" {
			return ScopedInventory{}, fmt.Errorf("nvidia-smi CSV row %d has missing required fields", rowNumber)
		}
		device := Device{
			UUID:              uuid,
			Name:              row[2],
			Architecture:      architectureForComputeCapability(row[3]),
			ComputeCapability: row[3],
			DriverVersion:     row[4],
			DriverCUDACeiling: ceiling,
			Observations:      []Observation{{Scope: scope, ScopeID: scopeID, Index: index}},
		}
		inventory.Devices = append(inventory.Devices, device)
		if inventory.DriverVersion == "" {
			inventory.DriverVersion = row[4]
		} else if inventory.DriverVersion != row[4] {
			return ScopedInventory{}, fmt.Errorf("inconsistent driver versions in %s inventory", scope)
		}
	}
	if len(inventory.Devices) == 0 {
		return ScopedInventory{}, fmt.Errorf("nvidia-smi returned no GPUs")
	}
	return inventory, nil
}

func architectureForComputeCapability(value string) string {
	major, minor, err := parseMajorMinor(value)
	if err != nil {
		return ""
	}
	switch major {
	case 6:
		return "Pascal"
	case 7:
		if minor == 0 || minor == 2 {
			return "Volta"
		}
		return "Turing"
	case 8:
		if minor == 9 {
			return "Ada"
		}
		return "Ampere"
	case 9:
		return "Hopper"
	default:
		return ""
	}
}

func parseDriverCUDACeiling(output []byte) (string, error) {
	var document struct {
		CUDAVersion string `xml:"cuda_version"`
	}
	if err := xml.Unmarshal(output, &document); err != nil {
		return "", fmt.Errorf("malformed nvidia-smi XML: %w", err)
	}
	document.CUDAVersion = strings.TrimSpace(document.CUDAVersion)
	if _, _, err := parseMajorMinor(document.CUDAVersion); err != nil {
		return "", fmt.Errorf("missing or invalid driver CUDA ceiling: %w", err)
	}
	return document.CUDAVersion, nil
}
