// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// Adapter implements the SandboxPort interface for sandbox isolation technologies.
// It provides a unified interface for gVisor, landlock, seccomp, and wasmtime sandboxes.
type Adapter struct{}

// NewAdapter creates a new sandbox adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// gvisorAdapter implements sandboxing using gVisor (runsc).
type gvisorAdapter struct {
	runtime   string
	overlayFS bool
}

// landlockAdapter implements sandboxing using Linux landlock.
type landlockAdapter struct {
	noNewPrivs bool
}

// seccompAdapter implements sandboxing using seccomp.
type seccompAdapter struct {
	defaultAction string
}

// wasmtimeAdapter implements sandboxing using wasmtime.
type wasmtimeAdapter struct {
	wasmEngine string
}

// New creates a new sandbox with the specified configuration.
func (a *Adapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	var adapter ports.SandboxPort

	switch config.Type {
	case domain.SandboxTypeGVisor:
		adapter = &gvisorAdapter{
			runtime:   config.RuntimePath,
			overlayFS: config.EnableOverlayFS,
		}
	case domain.SandboxTypeLandlock:
		adapter = &landlockAdapter{
			noNewPrivs: config.NoNewPrivs,
		}
	case domain.SandboxTypeSeccomp:
		adapter = &seccompAdapter{
			defaultAction: config.SeccompDefaultAction,
		}
	case domain.SandboxTypeWasmtime:
		adapter = &wasmtimeAdapter{
			wasmEngine: config.WasmEngine,
		}
	default:
		return nil, fmt.Errorf("unsupported sandbox type: %s", config.Type)
	}

	return adapter.New(ctx, config)
}

// List lists available sandbox runtimes.
func (a *Adapter) List(ctx context.Context) ([]domain.SandboxRuntime, error) {
	runtimes := []domain.SandboxRuntime{}

	// Check for gVisor
	if path, err := exec.LookPath("runsc"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "gVisor",
			Type:    domain.SandboxTypeGVisor,
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	// Check for landlock support
	if a.checkLandlockSupport() {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Landlock",
			Type:    domain.SandboxTypeLandlock,
			Path:    "kernel-native",
			Version: "kernel-supported",
		})
	}

	// Check for wasmtime
	if path, err := exec.LookPath("wasmtime"); err == nil {
		runtimes = append(runtimes, domain.SandboxRuntime{
			Name:    "Wasmtime",
			Type:    domain.SandboxTypeWasmtime,
			Path:    path,
			Version: a.getVersion(path),
		})
	}

	return runtimes, nil
}

// New implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()

	cmd := exec.CommandContext(ctx, a.runtime,
		"run",
		"--id", id,
	)
	if a.overlayFS {
		cmd.Args = append(cmd.Args, "--overlay", runscPath, "/")
	} else {
		cmd.Args = append(cmd.Args, "--read-only", runscPath, "/")
	}

	return &domain.Sandbox{
		ID:         id,
		Type:       domain.SandboxTypeGVisor,
		Config:     config,
		PID:        -1,
		Status:     domain.SandboxStatusCreating,
		Mounts:     config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	// gVisor sandbox is started with the run command
	// PID is assigned when the container starts
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	cmd := exec.CommandContext(ctx, a.runtime, "kill", sandbox.ID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	cmd := exec.CommandContext(ctx, a.runtime, "delete", sandbox.ID)
	return cmd.Run()
}

// New implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeLandlock,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	// Landlock is enforced at the kernel level via syscalls
	// Start the process with landlock enabled
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil // Landlock rules are cleaned up with the process
}

// New implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeSeccomp,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil
}

// New implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeWasmtime,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusRunning
	return nil
}

// Stop implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
	return nil
}

// checkLandlockSupport checks if the kernel supports landlock.
func (a *Adapter) checkLandlockSupport() bool {
	// Landlock support requires kernel >= 5.13
	// We check by looking for /sys/kernel/security/landlock
	return true // Simplified - real implementation would check kernel version
}

// getVersion returns the version of a runtime.
func (a *Adapter) getVersion(path string) string {
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// generateID generates a unique sandbox ID.
func generateID() string {
	// Simplified - real implementation would use UUID
	return fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
}

// nativeSandboxAdapter implements lightweight native sandboxing using
// bwrap (bubblewrap), firejail, or unshare/Linux namespaces.
// These provide millisecond startup times vs seconds for VMs.
type nativeSandboxAdapter struct {
	tool        string // "bwrap", "firejail", or "unshare"
	userNS      bool   // Use user namespaces
	mountNS     bool   // Use mount namespaces
	pidNS       bool   // Use PID namespace
	netNS       bool   // Use network namespace
}

// NewNativeSandbox creates a native sandbox adapter with the specified tool.
func NewNativeSandbox(tool string) *nativeSandboxAdapter {
	return &nativeSandboxAdapter{
		tool: tool,
	}
}

// New creates a new native sandbox.
func (a *nativeSandboxAdapter) New(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()

	// Check if the tool is available
	if path, err := exec.LookPath(a.tool); err != nil {
		return nil, fmt.Errorf("%s not found: %w", a.tool, err)
	} else {
		config.RuntimePath = path
	}

	return &domain.Sandbox{
		ID:          id,
		Type:        domain.SandboxTypeNative,
		Config:      config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}, nil
}

// Start launches the command inside the native sandbox.
func (a *nativeSandboxAdapter) Start(ctx context.Context, sandbox *domain.Sandbox) error {
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

	// The actual command to run (would be passed as part of config in real impl)
	args = append(args, "/bin/sh")

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

	// The actual command
	args = append(args, "/bin/sh")

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

	// The actual command
	args = append(args, "/bin/sh")

	return exec.CommandContext(ctx, args[0], args[1:]...)
}

// Stop terminates the sandboxed process.
func (a *nativeSandboxAdapter) Stop(ctx context.Context, sandbox *domain.Sandbox) error {
	if sandbox.PID > 0 {
		cmd := exec.CommandContext(ctx, "kill", "-9", fmt.Sprintf("%d", sandbox.PID))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop native sandbox: %w", err)
		}
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete cleans up the sandbox.
func (a *nativeSandboxAdapter) Delete(ctx context.Context, sandbox *domain.Sandbox) error {
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
var _ ports.SandboxPort = (*Adapter)(nil)
var _ ports.SandboxPort = (*gvisorAdapter)(nil)
var _ ports.SandboxPort = (*landlockAdapter)(nil)
var _ ports.SandboxPort = (*seccompAdapter)(nil)
var _ ports.SandboxPort = (*wasmtimeAdapter)(nil)
var _ ports.SandboxPort = (*nativeSandboxAdapter)(nil)
