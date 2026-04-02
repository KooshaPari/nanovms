package windows

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Adapter implements the ports.RuntimeAdapter for Windows/WSL2.
type Adapter struct {
	wslPath string
}

// New creates a new Windows/WSL2 adapter.
func New() (*Adapter, error) {
	wslPath, err := exec.LookPath("wsl.exe")
	if err != nil {
		return nil, fmt.Errorf("WSL2 not found: %w", err)
	}
	return &Adapter{wslPath: wslPath}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "windows-wsl2"
}

// Create creates a new WSL2 sandbox instance.
func (a *Adapter) Create(ctx context.Context, config interface{}) (string, error) {
	// Create a new WSL instance for the sandbox
	name := fmt.Sprintf("devenv-%s", config.(string))

	// Create WSL instance with Ubuntu
	cmd := exec.CommandContext(ctx, a.wslPath, "--", "ubuntu", "run", "echo", "Sandbox initialized")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create WSL sandbox: %w", err)
	}

	return name, nil
}

// Start starts an existing sandbox instance.
func (a *Adapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.wslPath, "--", "-d", id)
	return cmd.Run()
}

// Stop stops a running sandbox instance.
func (a *Adapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.wslPath, "--", "--terminate", id)
	return cmd.Run()
}

// Delete deletes a sandbox instance.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.wslPath, "--", "--unregister", id)
	return cmd.Run()
}

// Exec executes a command in the WSL sandbox.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	execCmd := exec.CommandContext(ctx, a.wslPath, "--", "-d", id, "--", "bash", "-c", strings.Join(cmd, " "))
	execCmd.Stdin = stdin
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	return execCmd.Run()
}

// Pull pulls an image (NOP for WSL - uses distributions instead).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	// WSL uses distributions, not images
	return nil
}
