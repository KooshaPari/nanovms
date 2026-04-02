package linux

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Adapter implements the ports.RuntimeAdapter for Linux native (namespace/cgroup).
type Adapter struct {
	rootless bool
}

// New creates a new Linux native adapter.
func New(rootless bool) *Adapter {
	return &Adapter{rootless: rootless}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	if a.rootless {
		return "linux-rootless"
	}
	return "linux-native"
}

// Create creates a new Linux sandbox using namespaces.
func (a *Adapter) Create(ctx context.Context, config interface{}) (string, error) {
	// For rootless, we use user namespaces
	// For privileged, we could use cgroups + namespaces
	name := fmt.Sprintf("devenv-%s", config.(string))

	// Create a new network namespace
	cmd := exec.CommandContext(ctx, "ip", "netns", "add", name)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create network namespace: %w", err)
	}

	return name, nil
}

// Start starts an existing sandbox.
func (a *Adapter) Start(ctx context.Context, id string) error {
	// Namespaces are started on creation
	return nil
}

// Stop stops a running sandbox.
func (a *Adapter) Stop(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "ip", "netns", "delete", id)
	return cmd.Run()
}

// Delete deletes a sandbox.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// Exec executes a command in the Linux sandbox.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	execCmd := exec.CommandContext(ctx, "ip", "netns", "exec", id, "bash", "-c", cmd[0])
	execCmd.Stdin = stdin
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	return execCmd.Run()
}

// Pull pulls an image (NOP - uses local filesystems).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	return nil
}
