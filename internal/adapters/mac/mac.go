// Package mac provides the macOS adapter using Lima/VZ.
package mac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kooshapari/devenv-abstraction/internal/domain"
	"github.com/kooshapari/devenv-abstraction/internal/ports"
)

// Adapter implements RuntimePort for macOS using Lima/VZ.
type Adapter struct {
	limaPath string
}

// NewAdapter creates a new macOS adapter.
func NewAdapter() (*Adapter, error) {
	// Check for lima installation
	limaPath, err := exec.LookPath("limactl")
	if err != nil {
		// Fallback to colima
		limaPath, err = exec.LookPath("colima")
		if err != nil {
			return nil, fmt.Errorf("neither lima nor colima found: %w", err)
		}
	}
	return &Adapter{limaPath: limaPath}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	if strings.Contains(a.limaPath, "colima") {
		return "colima"
	}
	return "lima"
}

// Create creates a new Lima VM.
func (a *Adapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	// Generate instance name
	name := config.Name
	if name == "" {
		name = fmt.Sprintf("devenv-%s", domain.GenerateID())
	}

	// Create Lima YAML config
	yamlConfig := a.generateLimaConfig(config)

	// Write config to temp file
	tmpPath := fmt.Sprintf("/tmp/devenv-%s.yaml", name)
	if err := writeFile(tmpPath, yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Create the VM
	cmd := exec.CommandContext(ctx, a.limaPath, "create", name, "--tty=false", "--vm-type=vz", "--vz-rosetta", "--volumes-from=devenv-templates")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("lima create failed: %w", err)
	}

	return &domain.Sandbox{
		ID:     name,
		Name:   name,
		Status: domain.StatusCreated,
		Config: config,
	}, nil
}

// Start starts the Lima VM.
func (a *Adapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.limaPath, "start", id)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}

// Stop stops the Lima VM.
func (a *Adapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.limaPath, "stop", id)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}

// Delete deletes the Lima VM.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.lima.Path, "delete", id, "--force")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}

// List lists all Lima VMs.
func (a *Adapter) List(ctx context.Context) ([]domain.Sandbox, error) {
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

// Status returns the status of a Lima VM.
func (a *Adapter) Status(ctx context.Context, id string) (domain.SandboxStatus, error) {
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

// Exec executes a command in the Lima VM.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	execCmd := exec.CommandContext(ctx, a.limaPath, "shell", id, "/bin/bash", "-c", strings.Join(cmd, " "))
	execCmd.Stdin = stdin
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	return execCmd.Run()
}

// Pull pulls an image (NOP for Lima - uses Lima templates instead).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	// Lima uses templates, not images
	return nil
}

// generateLimaConfig generates a Lima configuration for the sandbox.
func (a *Adapter) generateLimaConfig(config domain.SandboxConfig) string {
	return fmt.Sprintf(`images:
  - location: "https://example.com/fedoraLima.%s"
    arch: "aarch64"

cpus: %d
memory: %dG
disk: %dG

mounts:
  - location: "~"
    writable: %t
  - location: "/tmp/devenv"
    writable: true

environment:
  DEVENv_MODE: "%s"
`, config.OS, config.CPU, config.MemoryGB, config.DiskGB, config.ReadWrite, config.Mode)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
