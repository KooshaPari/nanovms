// Package sandbox / native.go — native sandbox adapter (bwrap, firejail, unshare).
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"github.com/kooshapari/nanovms/internal/domain"
)

type nativeSandboxAdapter struct {
	tool      string                     // "bwrap", "firejail", or "unshare"
	userNS    bool                       // Use user namespaces
	mountNS   bool                       // Use mount namespaces
	pidNS     bool                       // Use PID namespace
	netNS     bool                       // Use network namespace
	sandboxes map[string]*domain.Sandbox // Store sandboxes by ID
}

func NewNativeSandbox(tool string) *nativeSandboxAdapter {
	return &nativeSandboxAdapter{
		tool:      tool,
		sandboxes: make(map[string]*domain.Sandbox),
	}
}

func (a *nativeSandboxAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()

	// Check if the tool is available
	if path, err := exec.LookPath(a.tool); err != nil {
		return nil, fmt.Errorf("%s not found: %w", a.tool, err)
	} else {
		config.RuntimePath = path
	}

	sandbox := &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeNative,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}
	a.sandboxes[id] = sandbox
	return sandbox, nil
}

// Start launches the command inside the native sandbox.
func (a *nativeSandboxAdapter) Start(ctx context.Context, id string) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}

	var cmd *exec.Cmd

	switch a.tool {
	case "bwrap":
		cmd = a.startBwrap(ctx, sandbox)
	case "firejail":
		cmd = a.startFirejail(ctx, sandbox)
	case "unshare":
		cmd = a.startUnshare(ctx, sandbox)
	default:
		return fmt.Errorf("unsupported native sandbox tool: %s", a.tool)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start native sandbox: %w", err)
	}

	sandbox.PID = cmd.Process.Pid
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// resolveExecCommand returns the command-and-args vector to hand to the
// sandbox runtime. The user-supplied `config.NativeSandbox.Command` (if
// any) wins; otherwise we fall back to `/bin/sh` so the sandbox still
// launches a sensible default.
//
// Note: `config.Command` is the field on `NativeSandboxConfig`, not on
// `SandboxConfig` itself. The function tolerates a nil `config.NativeSandbox`
// and any other variant.
func resolveExecCommand(config *domain.SandboxConfig) []string {
	if config != nil && config.NativeSandbox != nil && len(config.NativeSandbox.Command) > 0 {
		return config.NativeSandbox.Command
	}
	return []string{"/bin/sh"}
}

// startBwrap starts a process using bubblewrap (bwrap).
func (a *nativeSandboxAdapter) startBwrap(ctx context.Context, sandbox *domain.Sandbox) *exec.Cmd {
	args := []string{"bwrap", "--share-net"} // Share network namespace

	// Add namespace flags
	if a.mountNS {
		args = append(args, "--unshare-mount")
	}
	if a.pidNS {
		args = append(args, "--unshare-pid")
	}
	if a.userNS {
		args = append(args, "--unshare-user")
	}

	// Read-only rootfs if specified
	if sandbox.Config.ReadOnlyRootfs {
		args = append(args, "--ro-bind", "/", "/")
	} else {
		args = append(args, "--bind", "/", "/")
	}

	// Add tmpfs for /tmp if specified
	if sandbox.Config.TmpfsTmp {
		args = append(args, "--tmpfs", "/tmp")
	}

	// Add bind mounts from config
	for _, mount := range sandbox.Mounts {
		if mount.ReadOnly {
			args = append(args, "--ro-bind", mount.Source, mount.Target)
		} else {
			args = append(args, "--bind", mount.Source, mount.Target)
		}
	}

	// Add seccomp if specified
	if sandbox.Config.SeccompProfile != "" {
		args = append(args, "--seccomp", sandbox.Config.SeccompProfile)
	}

	// Set working directory if specified
	if sandbox.Config.WorkDir != "" {
		args = append(args, "--chdir", sandbox.Config.WorkDir)
	}

	// The actual command (from config) — bwrap expects CMD after `--`.
	args = append(args, "--")
	args = append(args, resolveExecCommand(sandbox.Config)...)

	return exec.CommandContext(ctx, args[0], args[1:]...)
}

// startFirejail starts a process using firejail.
func (a *nativeSandboxAdapter) startFirejail(ctx context.Context, sandbox *domain.Sandbox) *exec.Cmd {
	args := []string{"firejail"}

	// Add namespace flags
	if !a.netNS {
		args = append(args, "--net=none")
	}
	if a.pidNS {
		args = append(args, "--private=pid")
	}

	// Add profile file if specified
	if sandbox.Config.FirejailProfile != "" {
		args = append(args, "--profile="+sandbox.Config.FirejailProfile)
	}

	// Add bind mounts from config
	for _, mount := range sandbox.Mounts {
		if mount.ReadOnly {
			args = append(args, "--read-only="+mount.Source)
		} else {
			args = append(args, "--bind="+mount.Source+"="+mount.Target)
		}
	}

	// The actual command (from config) — firejail takes the cmd after all flags.
	args = append(args, resolveExecCommand(sandbox.Config)...)

	return exec.CommandContext(ctx, args[0], args[1:]...)
}

// startUnshare starts a process using unshare with Linux namespaces.
func (a *nativeSandboxAdapter) startUnshare(ctx context.Context, sandbox *domain.Sandbox) *exec.Cmd {
	// Build unshare command
	args := []string{"unshare"}

	if a.userNS {
		args = append(args, "--user")
	}
	if a.mountNS {
		args = append(args, "--mount")
	}
	if a.pidNS {
		args = append(args, "--pid")
	}
	if a.netNS {
		// Note: --net requires CAP_NET_ADMIN
		args = append(args, "--net")
	}

	// Use fake root if user namespace
	if a.userNS {
		args = append(args, "--map-root-user")
	}

	// The actual command (from config).
	args = append(args, resolveExecCommand(sandbox.Config)...)

	return exec.CommandContext(ctx, args[0], args[1:]...)
}

// Stop terminates the sandboxed process.
func (a *nativeSandboxAdapter) Stop(ctx context.Context, id string, force bool) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	if sandbox.PID > 0 {
		signal := "SIGTERM"
		if force {
			signal = "SIGKILL"
		}
		cmd := exec.CommandContext(ctx, "kill", "-"+signal, fmt.Sprintf("%d", sandbox.PID))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop native sandbox: %w", err)
		}
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete cleans up the sandbox.
func (a *nativeSandboxAdapter) Delete(ctx context.Context, id string) error {
	// Remove from store
	delete(a.sandboxes, id)
	// Native sandboxes don't need cleanup - resources are freed when process exits
	return nil
}

// ListNativeSandboxes lists available native sandbox tools.
func (a *Adapter) ListNativeSandboxes(ctx context.Context) ([]domain.SandboxRuntime, error) {
	runtimes := []domain.SandboxRuntime{}

	// Check for bwrap
	if path, err := exec.LookPath("bwrap"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Bubblewrap (bwrap)",
			Type:    domain.SandboxTypeNative,
			SubType: "bwrap",
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	// Check for firejail
	if path, err := exec.LookPath("firejail"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Firejail",
			Type:    domain.SandboxTypeNative,
			SubType: "firejail",
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	// unshare is always available on Linux (part of util-linux)
	if path, err := exec.LookPath("unshare"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Linux Namespaces (unshare)",
			Type:    domain.SandboxTypeNative,
			SubType: "unshare",
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	return runtimes, nil
}

// Ensure ports.SandboxPort is implemented.
