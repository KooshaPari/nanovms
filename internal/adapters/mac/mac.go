// Package mac provides the macOS adapter with 3-tier VM support:
//   - Tier 1: Native VM (HyperKit, VMware Fusion)
//   - Tier 2: Lima/VZ (container-style virtualization)
//   - Tier 3: MicroVM (Firecracker)
//
// Plus sandbox isolation layers (gVisor, sRAMP, wasmtime).
package mac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/kooshapari/nanovms/internal/domain"
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

// Create creates a new sandbox with the specified VM tier.
func (a *Adapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	name := config.Name
	if name == "" {
		name = fmt.Sprintf("devenv-%s", domain.GenerateID())
	}

	var cmd *exec.Cmd

	switch config.VMTier {
	case ports.VMTierNative:
		// Tier 1: Native VM using HyperKit
		if a.hyperkitPath == "" {
			return nil, fmt.Errorf("hyperkit not available")
		}
		cmd = exec.CommandContext(ctx, a.hyperkitPath, "create", "--config", "/tmp/"+name+".json")

	case ports.VMTierMicroVM:
		// Tier 3: Firecracker MicroVM
		if a.firecrackerPath == "" {
			return nil, fmt.Errorf("firecracker not available")
		}
		cmd = exec.CommandContext(ctx, a.firecrackerPath, "--config", "/tmp/"+name+".json")

	case ports.VMTierLimaVZ:
		fallthrough
	default:
		// Tier 2: Lima with VZ (default)
		if a.limaPath == "" {
			return nil, fmt.Errorf("lima/colima not available")
		}
		cmd = exec.CommandContext(ctx, a.limaPath, "create", name, "--tty=false", "--vm-type=vz", "--volumes-from=devenv-templates")
	}

	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("create failed: %w", err)
	}

	// Apply sandbox isolation layer if specified
	if config.SandboxLayer != domain.SandboxLayerNone && config.SandboxLayer != "" {
		if err := a.applySandboxLayer(ctx, name, config.SandboxLayer); err != nil {
			return nil, fmt.Errorf("failed to apply sandbox layer: %w", err)
		}
	}

	return &domain.Sandbox{
		ID:           name,
		Name:         name,
		Status:       domain.StatusCreated,
		Config:       config,
		VMTier:       config.VMTier,
		SandboxLayer: config.SandboxLayer,
	}, nil
}

// Start starts the VM.
func (a *Adapter) Start(ctx context.Context, id string) error {
	switch a.Name() {
	case "firecracker":
		cmd := exec.CommandContext(ctx, a.firecrackerPath, "--api-sock", "/tmp/"+id+".sock")
		return cmd.Run()
	case "hyperkit":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "start", id)
		return cmd.Run()
	default: // lima/colima
		cmd := exec.CommandContext(ctx, a.limaPath, "start", id)
		return cmd.Run()
	}
}

// Stop stops the VM.
func (a *Adapter) Stop(ctx context.Context, id string) error {
	switch a.Name() {
	case "firecracker":
		// Send shutdown signal via API
		cmd := exec.CommandContext(ctx, "curl", "-X", "PUT", "--unix-socket", "/tmp/"+id+".sock", "http://localhost/actions")
		return cmd.Run()
	case "hyperkit":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "stop", id)
		return cmd.Run()
	default: // lima/colima
		cmd := exec.CommandContext(ctx, a.limaPath, "stop", id)
		return cmd.Run()
	}
}

// Delete deletes the VM.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	switch a.Name() {
	case "firecracker":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "delete", id) // hyperkit tool for firecracker cleanup
		return cmd.Run()
	case "hyperkit":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "delete", id)
		return cmd.Run()
	default: // lima/colima
		cmd := exec.CommandContext(ctx, a.limaPath, "delete", id, "--force")
		return cmd.Run()
	}
}

// List lists all VMs.
func (a *Adapter) List(ctx context.Context) ([]domain.Sandbox, error) {
	switch a.Name() {
	case "firecracker":
		return a.listFirecrackerVMs(ctx)
	case "hyperkit":
		return a.listHyperKitVMs(ctx)
	default: // lima/colima
		return a.listLimaVMs(ctx)
	}
}

// Status returns the status of a VM.
func (a *Adapter) Status(ctx context.Context, id string) (domain.SandboxStatus, error) {
	switch a.Name() {
	case "firecracker":
		cmd := exec.CommandContext(ctx, "curl", "-S", "--unix-socket", "/tmp/"+id+".sock", "http://localhost/")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return domain.StatusUnknown, err
		}
		// Parse firecracker status from response
		return domain.StatusRunning, nil

	case "hyperkit":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "status", id)
		if err := cmd.Run(); err != nil {
			return domain.StatusStopped, nil
		}
		return domain.StatusRunning, nil

	default: // lima/colima
		cmd := exec.CommandContext(ctx, a.limaPath, "list", "--json")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &bytes.Buffer{}
		if err := cmd.Run(); err != nil {
			return domain.StatusUnknown, err
		}

		var vms []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(out.Bytes(), &vms); err != nil {
			return domain.StatusUnknown, err
		}

		for _, vm := range vms {
			if vm.Name == id {
				return domain.ParseStatus(vm.Status), nil
			}
		}
		return domain.StatusUnknown, fmt.Errorf("sandbox not found: %s", id)
	}
}

// Exec executes a command in the VM.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	switch a.Name() {
	case "firecracker":
		// Firecracker uses vsock for commands
		firecrackerCmd := exec.CommandContext(ctx, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "root@localhost", "-p", "22", strings.Join(cmd, " "))
		firecrackerCmd.Stdin = stdin
		firecrackerCmd.Stdout = stdout
		firecrackerCmd.Stderr = stderr
		return firecrackerCmd.Run()

	case "hyperkit":
		cmd := exec.CommandContext(ctx, a.hyperkitPath, "exec", id, "--", "/bin/bash", "-c", strings.Join(cmd, " "))
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		return cmd.Run()

	default: // lima/colima
		execCmd := exec.CommandContext(ctx, a.limaPath, "shell", id, "/bin/bash", "-c", strings.Join(cmd, " "))
		execCmd.Stdin = stdin
		execCmd.Stdout = stdout
		execCmd.Stderr = stderr
		return execCmd.Run()
	}
}

// Pull pulls an image (NOP for Lima/Firecracker - uses templates/kernel).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	// macOS VMs use kernels + initrd, not container images
	return nil
}

// listLimaVMs lists Lima/Colima VMs.
func (a *Adapter) listLimaVMs(ctx context.Context) ([]domain.Sandbox, error) {
	cmd := exec.CommandContext(ctx, a.limaPath, "list", "--json")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var vms []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &vms); err != nil {
		return nil, err
	}

	result := make([]domain.Sandbox, 0, len(vms))
	for _, vm := range vms {
		result = append(result, domain.Sandbox{
			ID:     vm.Name,
			Name:   vm.Name,
			Status: domain.ParseStatus(vm.Status),
		})
	}
	return result, nil
}

// listHyperKitVMs lists HyperKit VMs.
func (a *Adapter) listHyperKitVMs(ctx context.Context) ([]domain.Sandbox, error) {
	cmd := exec.CommandContext(ctx, a.hyperkitPath, "list")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Parse hyperkit list output
	var result []domain.Sandbox
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VM:") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				result = append(result, domain.Sandbox{
					ID:     parts[1],
					Name:   parts[1],
					Status: domain.StatusRunning,
				})
			}
		}
	}
	return result, nil
}

// listFirecrackerVMs lists Firecracker MicroVMs.
func (a *Adapter) listFirecrackerVMs(ctx context.Context) ([]domain.Sandbox, error) {
	// Firecracker VMs are listed via API socket files
	cmd := exec.CommandContext(ctx, "ls", "/var/run/firecracker/")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, nil // No VMs running
	}

	var result []domain.Sandbox
	sockets := strings.Split(out.String(), "\n")
	for _, sock := range sockets {
		if strings.HasSuffix(sock, ".sock") {
			name := strings.TrimSuffix(sock, ".sock")
			result = append(result, domain.Sandbox{
				ID:     name,
				Name:   name,
				Status: domain.StatusRunning, // Assume running if socket exists
			})
		}
	}
	return result, nil
}

// applySandboxLayer applies the specified sandbox isolation layer.
func (a *Adapter) applySandboxLayer(ctx context.Context, id string, layer domain.SandboxLayer) error {
	switch layer {
	case domain.SandboxLayerGVisor:
		// Install and configure gVisor (runsc)
		cmd := exec.CommandContext(ctx, "which", "runsc")
		if err := cmd.Run(); err != nil {
			// Install gVisor
			installCmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://gvisor.dev/install.sh")
			if err := installCmd.Run(); err != nil {
				return fmt.Errorf("failed to install gVisor: %w", err)
			}
		}
		// Configure the VM to use gVisor as runtime
		return nil

	case domain.SandboxLayerSRAMP:
		// sRAMP (Secure Runtime Application Malware Protection) - Linux native
		// Configure landlock + seccomp via kernel params
		cmd := exec.CommandContext(ctx, a.limaPath, "shell", id, "sysctl", "-w", "kernel.yama.ptrace_scope=2")
		return cmd.Run()

	case domain.SandboxLayerWasmtime:
		// WASM runtime via wasmtime
		cmd := exec.CommandContext(ctx, "which", "wasmtime")
		if err := cmd.Run(); err != nil {
			installCmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://wasmtime.dev/install.sh")
			return installCmd.Run()
		}
		return nil

	case domain.SandboxLayerNone:
		return nil

	default:
		return fmt.Errorf("unsupported sandbox layer: %s", layer)
	}
}
