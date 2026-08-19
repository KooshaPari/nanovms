// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- Firecracker adapter for microVM-based isolation.

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// FirecrackerConfig extends SandboxConfig with Firecracker-specific options.
type FirecrackerConfig struct {
	SocketPath string
	KernelPath string
	RootFSPath string
	VCPUs      int
	MemSizeMB  int
	HTTPEndpoint string
}

// FirecrackerAdapter manages microVMs using the firecracker binary.
type FirecrackerAdapter struct {
	mu       sync.Mutex
	binary   string
	registry map[SandboxID]*firecrackerVM
}

type firecrackerVM struct {
	id     SandboxID
	config SandboxConfig
	fcConf FirecrackerConfig
	status SandboxStatus
	cmd    *exec.Cmd
}

// NewFirecrackerAdapter creates a new Firecracker adapter.
func NewFirecrackerAdapter() *FirecrackerAdapter {
	binary := "firecracker"
	if p, err := exec.LookPath("firecracker"); err == nil {
		binary = p
	}
	return &FirecrackerAdapter{
		binary:   binary,
		registry: make(map[SandboxID]*firecrackerVM),
	}
}

// Name returns the adapter name.
func (f *FirecrackerAdapter) Name() string {
	return "firecracker"
}

// IsolationLevel returns the isolation level for Firecracker microVMs.
func (f *FirecrackerAdapter) IsolationLevel() string {
	return "vm"
}

// Create provisions a new Firecracker microVM.
func (f *FirecrackerAdapter) Create(ctx context.Context, cfg SandboxConfig) (*SandboxResult, error) {
	id := SandboxID(fmt.Sprintf("fc-%s-%d", cfg.Name, 0))

	socketPath := filepath.Join(os.TempDir(), string(id)+".sock")
	fcConf := FirecrackerConfig{
		SocketPath: socketPath,
		VCPUs:      1,
		MemSizeMB:  256,
	}

	f.mu.Lock()
	f.registry[id] = &firecrackerVM{
		id:     id,
		config: cfg,
		fcConf: fcConf,
		status: SandboxStatusCreated,
	}
	f.mu.Unlock()

	return &SandboxResult{ID: id, Status: SandboxStatusCreated}, nil
}

// Stop halts the Firecracker microVM by sending SIGTERM or SIGKILL.
func (f *FirecrackerAdapter) Stop(ctx context.Context, id SandboxID, force bool) error {
	f.mu.Lock()
	vm, ok := f.registry[id]
	f.mu.Unlock()
	if !ok {
		return fmt.Errorf("firecracker VM not found: %s", id)
	}

	if vm.cmd != nil && vm.cmd.Process != nil {
		signal := "SIGTERM"
		if force {
			signal = "SIGKILL"
		}
		killCmd := exec.CommandContext(ctx, "kill", "-"+signal, fmt.Sprintf("%d", vm.cmd.Process.Pid))
		_ = killCmd.Run()
		_ = vm.cmd.Wait()
	}

	// Remove the API socket
	_ = os.Remove(vm.fcConf.SocketPath)

	f.mu.Lock()
	vm.status = SandboxStatusStopped
	f.mu.Unlock()

	return nil
}

// Remove tears down the microVM and cleans up resources.
func (f *FirecrackerAdapter) Remove(ctx context.Context, id SandboxID, force bool) error {
	_ = f.Stop(ctx, id, force)

	f.mu.Lock()
	delete(f.registry, id)
	f.mu.Unlock()

	return nil
}

// Status queries the status of a Firecracker microVM.
func (f *FirecrackerAdapter) Status(ctx context.Context, id SandboxID) (*SandboxResult, error) {
	f.mu.Lock()
	vm, ok := f.registry[id]
	f.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("firecracker VM not found: %s", id)
	}

	status := vm.status
	if vm.cmd != nil && vm.cmd.Process != nil {
		// Check if process is still alive
		pidFile := fmt.Sprintf("/proc/%d/status", vm.cmd.Process.Pid)
		if _, err := os.Stat(pidFile); err == nil {
			status = SandboxStatusRunning
		} else {
			status = SandboxStatusStopped
		}
	}

	return &SandboxResult{ID: id, Status: status}, nil
}

// Execute runs a command inside the Firecracker microVM via jailer nsenter.
func (f *FirecrackerAdapter) Execute(ctx context.Context, id SandboxID, cmdArgs []string) (string, error) {
	f.mu.Lock()
	vm, ok := f.registry[id]
	f.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("firecracker VM not found: %s", id)
	}
	if vm.cmd == nil || vm.cmd.Process == nil || vm.cmd.Process.Pid <= 0 {
		return "", fmt.Errorf("firecracker VM %s is not running", id)
	}

	pid := fmt.Sprintf("%d", vm.cmd.Process.Pid)
	args := []string{"-t", pid, "-m", "-u", "-i", "-p", "--"}
	args = append(args, cmdArgs...)

	cmd := exec.CommandContext(ctx, "nsenter", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("nsenter exec failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return string(out), nil
}
