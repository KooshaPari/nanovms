// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// cryptoRandReader is the global random reader used for ID generation.
var cryptoRandReader io.Reader = rand.Reader

// runscPath is the path to the runsc binary (gVisor runtime).
var runscPath = "/usr/local/bin/runsc"

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
		ID:          id,
		Type:        domain.SandboxTypeGVisor,
		Config:      &config,
		PID:         -1,
		Status:      domain.SandboxStatusCreating,
		Mounts:      config.Mounts,
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
		Config:      &config,
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
		Config:      &config,
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
		Config:      &config,
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
// Landlock requires kernel >= 5.13 (released 2021-06-27).
// We verify by checking the /sys/kernel/security/landlock syscall entry.
func (a *Adapter) checkLandlockSupport() bool {
	// Check for landlock syscall support via /proc/sys/kernel/unprivileged_userns_clone
	// and verify kernel version >= 5.13.
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	version := strings.TrimSpace(string(data))
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major := 0
	minor := 0
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	if major > 5 || (major == 5 && minor >= 13) {
		// Also verify landlock filesystem entry exists
		_, err := os.Stat("/sys/kernel/security/landlock")
		return err == nil
	}
	return false
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

// logsForSandbox retrieves logs for a running sandbox by PID.
func logsForSandbox(ctx context.Context, sb *domain.Sandbox) (io.ReadCloser, error) {
	if sb == nil {
		return nil, fmt.Errorf("sandbox is nil")
	}
	if sb.PID <= 0 {
		return nil, fmt.Errorf("sandbox PID not available (sandbox may not be running)")
	}
	// Use journald to retrieve logs for the sandbox unit, or fall back to /proc/{pid}/
	args := []string{"journalctl", "--no-pager", "-p", "info"}
	args = append(args, []string{"_PID=" + fmt.Sprintf("%d", sb.PID)}...)
	args = append(args, []string{"--since", "1 hour ago"}...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}
	return r, nil
}

// execInSandbox executes a command inside a running sandbox's namespace via nsenter.
func execInSandbox(ctx context.Context, sb *domain.Sandbox, cmdArgs []string) (io.ReadCloser, error) {
	if sb == nil {
		return nil, fmt.Errorf("sandbox is nil")
	}
	if sb.PID <= 0 {
		return nil, fmt.Errorf("sandbox PID not available (sandbox may not be running)")
	}
	args := []string{"nsenter", "-t", fmt.Sprintf("%d", sb.PID), "-m", "-u", "-i", "-p", "--"}
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	cmd.Stderr = cmd.Stdout // Merge stderr into stdout for single stream
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start nsenter: %w", err)
	}
	return r, nil
}

// metricsForSandbox collects CPU and memory metrics for a running sandbox process.
func metricsForSandbox(ctx context.Context, sb *domain.Sandbox) (*domain.SandboxMetrics, error) {
	if sb == nil {
		return nil, fmt.Errorf("sandbox is nil")
	}
	metrics := &domain.SandboxMetrics{SandboxID: sb.ID}
	if sb.PID <= 0 {
		return metrics, nil
	}
	// Read CPU and memory from /proc/{pid}/status and /proc/{pid}/stat
	statusData, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", sb.PID))
	if err != nil {
		return metrics, nil // Process may have exited
	}
	for _, line := range strings.Split(string(statusData), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			var kb int
			fmt.Sscanf(line, "VmRSS: %d kB", &kb)
			metrics.MemoryUsage = int64(kb) * 1024
		}
	}
	statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", sb.PID))
	if err != nil {
		return metrics, nil
	}
	// Fields: pid comm state ppid ... utime stime (14,15) ... clock tick
	fields := strings.Split(string(statData), " ")
	if len(fields) > 21 {
		var utime, stime int64
		fmt.Sscanf(fields[13], "%d", &utime)
		fmt.Sscanf(fields[14], "%d", &stime)
		// Convert clock ticks to percentage (simplified: %CPU = (utime+stime)/CLK_TCK)
		metrics.CPUUsage = float64(utime+stime) / 100.0 // 100 ticks/sec assumption
	}
	return metrics, nil
}

// generateID generates a cryptographically random UUID-based sandbox ID.
func generateID() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(cryptoRandReader, b); err != nil {
		// Fallback to nanoseconds + PID if crypto/rand fails (should not happen)
		return fmt.Sprintf("sandbox-%d-%d", time.Now().UnixNano(), os.Getpid())
	}
	return fmt.Sprintf("sandbox-%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// nativeSandboxAdapter implements lightweight native sandboxing using
// bwrap (bubblewrap), firejail, or unshare/Linux namespaces.
// These provide millisecond startup times vs seconds for VMs.
type nativeSandboxAdapter struct {
	tool      string                     // "bwrap", "firejail", or "unshare"
	userNS    bool                       // Use user namespaces
	mountNS   bool                       // Use mount namespaces
	pidNS     bool                       // Use PID namespace
	netNS     bool                       // Use network namespace
	sandboxes map[string]*domain.Sandbox // Store sandboxes by ID
}

// NewNativeSandbox creates a native sandbox adapter with the specified tool.
func NewNativeSandbox(tool string) *nativeSandboxAdapter {
	return &nativeSandboxAdapter{
		tool:      tool,
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
	result := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, sb := range a.sandboxes {
		result = append(result, sb)
	}
	return result, nil
}

// Get implements ports.SandboxPort for Adapter.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return sb, nil
}

// Logs implements ports.SandboxPort for Adapter.
// Returns logs by delegating to the native sandbox adapter if available,
// or by querying the runtime's log mechanism.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return logsForSandbox(ctx, sb)
}

// Exec implements ports.SandboxPort for Adapter.
// Executes a command in the specified sandbox using the native adapter.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return execInSandbox(ctx, sb, cmd)
}

// Metrics implements ports.SandboxPort for Adapter.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return metricsForSandbox(ctx, sb)
}

// List implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for gvisorAdapter.
// Note: gvisorAdapter does not maintain local sandbox storage.
// The caller must track sandbox IDs and re-create the adapter as needed.
func (a *gvisorAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Verify the sandbox still exists by querying runc
	cmd := exec.CommandContext(ctx, a.runtime, "ps", id)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	// Construct a minimal sandbox reference (PIDs tracked externally)
	return &domain.Sandbox{
		ID:     id,
		Status: domain.SandboxStatusRunning,
		Type:   domain.SandboxTypeGVisor,
	}, nil
}

// Logs implements ports.SandboxPort for gvisorAdapter.
// Retrieves logs via runsc log command or journald.
func (a *gvisorAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{runscPath, "logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, id)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "GvisorRuntime="+a.runtime)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runsc logs: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for gvisorAdapter.
// Executes a command in a running gVisor sandbox via runsc exec.
func (a *gvisorAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	args := []string{runscPath, "exec"}
	args = append(args, id)
	args = append(args, "--")
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "GvisorRuntime="+a.runtime)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start runsc exec: %w", err)
	}
	return r, nil
}

// Metrics implements ports.SandboxPort for gvisorAdapter.
func (a *gvisorAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	metrics := &domain.SandboxMetrics{SandboxID: id}
	// Query runc for process stats
	cmd := exec.CommandContext(ctx, a.runtime, "ps", id)
	out, err := cmd.Output()
	if err != nil {
		return metrics, nil // Return empty metrics if query fails
	}
	// Parse output (simplified - production would parse full ps output)
	if len(out) > 0 {
		metrics.CPUUsage = 0 // Would be parsed from runc stats
	}
	return metrics, nil
}

// List implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for landlockAdapter.
// Landlock is enforced at the kernel level; sandboxes are tracked by their PIDs.
// Use the PID stored when the sandboxed process was started.
func (a *landlockAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Landlock sandboxes are tracked via PID files or external process management.
	// Return a placeholder; real implementation would read PID from a tracking file.
	_ = a.noNewPrivs // suppress unused warning
	return nil, fmt.Errorf("landlock sandbox must be tracked externally by PID for id=%s", id)
}

// Logs implements ports.SandboxPort for landlockAdapter.
// Landlock sandboxes write logs via the container runtime or journald.
func (a *landlockAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"journalctl", "-t", "landlock-" + id}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for landlockAdapter.
// Executes in the landlock sandbox by finding the process and using nsenter.
func (a *landlockAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	// Landlock sandboxes must be tracked externally; exec via nsenter.
	// For now, return not implemented with guidance.
	_ = a.noNewPrivs
	return nil, fmt.Errorf("landlock sandbox exec requires external PID tracking; use nsenter with known PID for id=%s", id)
}

// Metrics implements ports.SandboxPort for landlockAdapter.
func (a *landlockAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.noNewPrivs
	return &domain.SandboxMetrics{SandboxID: id}, nil
}

// List implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for seccompAdapter.
// Seccomp sandboxes are enforced at the process level; tracked externally.
func (a *seccompAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// Seccomp is applied via prctl; sandbox state is managed by the parent process.
	return nil, fmt.Errorf("seccomp sandbox tracked externally; use container runtime for id=%s", id)
}

// Logs implements ports.SandboxPort for seccompAdapter.
// Seccomp sandboxes log via the controlling runtime (runc, containerd, etc.).
func (a *seccompAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"journalctl", "-t", "seccomp-" + id}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}
	return r, nil
}

// Exec implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	_ = a.defaultAction
	return nil, fmt.Errorf("seccomp sandbox exec requires container runtime; use runc exec for id=%s", id)
}

// Metrics implements ports.SandboxPort for seccompAdapter.
func (a *seccompAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.defaultAction
	return &domain.SandboxMetrics{SandboxID: id}, nil
}

// List implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	return []*domain.Sandbox{}, nil
}

// Get implements ports.SandboxPort for wasmtimeAdapter.
// wasmtimeAdapter does not maintain sandbox state; modules are stateless WASM instances.
func (a *wasmtimeAdapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	// WASM instances are stateless; module state is in the running process.
	// The caller should track WASM module IDs separately.
	return nil, fmt.Errorf("WASM sandbox state tracked by caller; use wasmtime instance API for id=%s", id)
}

// Logs implements ports.SandboxPort for wasmtimeAdapter.
// WASM modules do not produce traditional logs; stderr is captured via wasmtime.
func (a *wasmtimeAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	// WASM stderr is redirected by the calling runtime.
	// Use the wasmtime --env flag to capture stderr, or check the calling process.
	_ = a.wasmEngine
	return nil, fmt.Errorf("WASM logs must be captured by the calling runtime; use wasmtime with --dir for id=%s", id)
}

// Exec implements ports.SandboxPort for wasmtimeAdapter.
// Executes a WASM module via wasmtime with the given command-line arguments.
func (a *wasmtimeAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	args := []string{"wasmtime"}
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start wasmtime: %w", err)
	}
	return r, nil
}

// Metrics implements ports.SandboxPort for wasmtimeAdapter.
func (a *wasmtimeAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	_ = a.wasmEngine
	return &domain.SandboxMetrics{SandboxID: id}, nil
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
// Retrieves logs from a running native sandbox via journald.
func (a *nativeSandboxAdapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return logsForSandbox(ctx, sb)
}

// Exec implements ports.SandboxPort for nativeSandboxAdapter.
// Executes a command in the native sandbox's namespace via nsenter.
func (a *nativeSandboxAdapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	sb, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return execInSandbox(ctx, sb, cmdArgs)
}

// Metrics implements ports.SandboxPort for nativeSandboxAdapter.
func (a *nativeSandboxAdapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	sandbox, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return &domain.SandboxMetrics{
		SandboxID:   sandbox.ID,
		CPUUsage:    0,
		MemoryUsage: 0,
	}, nil
}
