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
		Config:       &config,
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
