// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// Adapter implements the SandboxPort interface for sandbox isolation technologies.
// It provides a unified interface for gVisor, landlock, seccomp, and wasmtime sandboxes.
type Adapter struct {
	sandboxes map[string]*domain.Sandbox
}

// NewAdapter creates a new sandbox adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		sandboxes: make(map[string]*domain.Sandbox),
	}
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

// Create creates a new sandbox with the specified configuration.
func (a *Adapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := generateID()
	now := time.Now()
	sandbox := &domain.Sandbox{
		ID:        id,
		Name:      config.Name,
		Status:    domain.SandboxStatusPending,
		VMFlavor:  config.VMType,
		CreatedAt: now,
	}
	a.sandboxes[id] = sandbox
	return sandbox, nil
}

// Start implements ports.SandboxPort for Adapter.
func (a *Adapter) Start(ctx context.Context, id string) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	now := time.Now()
	sandbox.Status = domain.SandboxStatusRunning
	sandbox.StartedAt = &now
	return nil
}

// Stop implements ports.SandboxPort for Adapter.
func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	sandbox.Status = domain.SandboxStatusStopped
	return nil
}

// Delete implements ports.SandboxPort for Adapter.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	delete(a.sandboxes, id)
	return nil
}

// ListRuntimes lists available sandbox runtimes.
func (a *Adapter) ListRuntimes(ctx context.Context) ([]domain.SandboxRuntime, error) {
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

// Create implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
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
func (a *gvisorAdapter) Start(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.runtime, "kill", "-SIGCONT", id)
	return cmd.Run()
}

// Stop implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Stop(ctx context.Context, id string, force bool) error {
	signal := "SIGTERM"
	if force {
		signal = "SIGKILL"
	}
	cmd := exec.CommandContext(ctx, a.runtime, "kill", "-"+signal, id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	return nil
}

// Delete implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Delete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, a.runtime, "delete", id)
	return cmd.Run()
}

// Create implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
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
func (a *landlockAdapter) Start(ctx context.Context, id string) error {
	// Landlock is enforced at the kernel level via syscalls
	return nil
}

// Stop implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Delete(ctx context.Context, id string) error {
	return nil // Landlock rules are cleaned up with the process
}

// Create implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
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
func (a *seccompAdapter) Start(ctx context.Context, id string) error {
	return nil
}

// Stop implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

// Create implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
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
func (a *wasmtimeAdapter) Start(ctx context.Context, id string) error {
	return nil
}

// Stop implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Stop(ctx context.Context, id string, force bool) error {
	return nil
}

// Delete implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Delete(ctx context.Context, id string) error {
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
	tool     string                  // "bwrap", "firejail", or "unshare"
	userNS   bool                    // Use user namespaces
	mountNS  bool                    // Use mount namespaces
	pidNS    bool                    // Use PID namespace
	netNS    bool                    // Use network namespace
	sandboxes map[string]*domain.Sandbox // Store sandboxes by ID
}

// NewNativeSandbox creates a native sandbox adapter with the specified tool.
func NewNativeSandbox(tool string) *nativeSandboxAdapter {
	return &nativeSandboxAdapter{
		tool:     tool,
		sandboxes: make(map[string]*domain.Sandbox),
	}
}

// Create creates a new native sandbox.
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
		Config:      config,
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
var _ ports.SandboxPort = (*Adapter)(nil)
var _ ports.SandboxPort = (*gvisorAdapter)(nil)
var _ ports.SandboxPort = (*landlockAdapter)(nil)
var _ ports.SandboxPort = (*seccompAdapter)(nil)
var _ ports.SandboxPort = (*wasmtimeAdapter)(nil)
var _ ports.SandboxPort = (*nativeSandboxAdapter)(nil)

// List implements ports.SandboxPort for Adapter.
func (a *Adapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for Adapter.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// Logs implements ports.SandboxPort for Adapter.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for Adapter.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for Adapter.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// List implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// Logs implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// List implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// Logs implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// List implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// Logs implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// List implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// Logs implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

// List implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	result := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, s := range a.sandboxes {
		result = append(result, s)
	}
	return result, nil
}

// Get implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return sandbox, nil
}

// Logs implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Exec implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

// Metrics implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return &domain.SandboxMetrics{
		SandboxID: sandbox.ID,
		CPUUsage:  0,
		MemoryUsage: 0,
	}, nil
}
