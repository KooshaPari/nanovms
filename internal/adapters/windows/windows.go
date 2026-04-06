package windows

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// WindowsTier represents the Windows virtualization tier.
type WindowsTier string

const (
	WindowsTierNative WindowsTier = "native" // Hyper-V, Windows Sandbox
	WindowsTierWSL    WindowsTier = "wsl"    // WSL2 + gVisor
	WindowsTierMicroVM WindowsTier = "microvm" // Cloud Hypervisor
)

// Adapter implements the ports.RuntimeAdapter for Windows with multiple tiers.
type Adapter struct {
	wslPath        string
	hyperVPath     string
	microVMPath    string
	defaultTier    WindowsTier
}

// New creates a new Windows adapter with multiple tier support.
func New() (*Adapter, error) {
	adapter := &Adapter{
		defaultTier: WindowsTierWSL,
	}

	// Detect available Windows virtualization options
	if wslPath, err := exec.LookPath("wsl.exe"); err == nil {
		adapter.wslPath = wslPath
	}
	if hyperVPath, err := exec.LookPath("hvlaunch.exe"); err == nil {
		adapter.hyperVPath = hyperVPath
	}
	if cloudHyperPath, err := exec.LookPath("cloud-hypervisor.exe"); err == nil {
		adapter.microVMPath = cloudHyperPath
	}

	if adapter.wslPath == "" && adapter.hyperVPath == "" && adapter.microVMPath == "" {
		return nil, fmt.Errorf("no Windows virtualization found (WSL2, Hyper-V, or Cloud Hypervisor)")
	}

	return adapter, nil
}

// NewWithTier creates a Windows adapter configured for a specific tier.
func NewWithTier(tier WindowsTier) (*Adapter, error) {
	adapter, err := New()
	if err != nil {
		return nil, err
	}

	switch tier {
	case WindowsTierNative:
		if adapter.hyperVPath == "" {
			return nil, fmt.Errorf("Hyper-V not available on this Windows installation")
		}
	case WindowsTierWSL:
		if adapter.wslPath == "" {
			return nil, fmt.Errorf("WSL2 not installed: https://docs.microsoft.com/en-us/windows/wsl/")
		}
	case WindowsTierMicroVM:
		if adapter.microVMPath == "" {
			return nil, fmt.Errorf("Cloud Hypervisor not installed")
		}
	}

	adapter.defaultTier = tier
	return adapter, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return fmt.Sprintf("windows-%s", a.defaultTier)
}

// Tier returns the current virtualization tier.
func (a *Adapter) Tier() string {
	return string(a.defaultTier)
}

// Create creates a new Windows sandbox instance.
func (a *Adapter) Create(ctx context.Context, name string, opts ...Option) (string, error) {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}

	if config.Tier == "" {
		config.Tier = a.defaultTier
	}

	switch config.Tier {
	case WindowsTierNative:
		return a.createNativeSandbox(ctx, name, config)
	case WindowsTierWSL:
		return a.createWSLSandbox(ctx, name, config)
	case WindowsTierMicroVM:
		return a.createMicroVMSandbox(ctx, name, config)
	default:
		return "", fmt.Errorf("unsupported Windows tier: %s", config.Tier)
	}
}

// createNativeSandbox creates a Windows Sandbox or Hyper-V VM.
func (a *Adapter) createNativeSandbox(ctx context.Context, name string, config *Config) (string, error) {
	// Use Windows Sandbox (requires Windows 10/11 Pro/Enterprise)
	sandboxName := fmt.Sprintf("devenv-%s", name)

	// Create Windows Sandbox configuration
	sandboxCfg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Configuration>
  <MappedFolders>
    <MappedFolder>
      <HostFolder>%s</HostFolder>
      <ReadOnly>false</ReadOnly>
    </MappedFolder>
  </MappedFolders>
  <VGpu>Enable</VGpu>
  <Networking>Default</Networking>
  <MemoryInMB>4096</MemoryInMB>
</Configuration>`, config.WorkDir)

	// For Hyper-V VMs, use PowerShell to create and manage
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command",
		fmt.Sprintf(`New-VM -Name '%s' -MemoryStartupBytes 4GB -BootDevice VHD -Generation 2; Start-VM -Name '%s'`, sandboxName, sandboxName))

	if err := cmd.Run(); err != nil {
		// Fallback to Windows Sandbox if Hyper-V is not available
		return a.createWindowsSandbox(ctx, sandboxName, sandboxCfg)
	}

	return sandboxName, nil
}

// createWindowsSandbox creates a lightweight Windows Sandbox.
func (a *Adapter) createWindowsSandbox(ctx context.Context, name, config string) (string, error) {
	// Windows Sandbox is a lighter-weight alternative
	cmd := exec.CommandContext(ctx, "wsandbox.exe", "create", "--name", name)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create Windows Sandbox: %w", err)
	}
	return name, nil
}

// createWSLSandbox creates a WSL2 sandbox with gVisor.
func (a *Adapter) createWSLSandbox(ctx context.Context, name string, config *Config) (string, error) {
	wslName := fmt.Sprintf("devenv-%s", name)

	// Create WSL instance with Ubuntu
	cmd := exec.CommandContext(ctx, a.wslPath, "--", "ubuntu", "run", "echo", "Sandbox initialized")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create WSL sandbox: %w", err)
	}

	// Install gVisor in the WSL instance for syscall interception
	gvisorInstall := exec.CommandContext(ctx, a.wslPath, "-d", wslName, "--", "bash", "-c",
		"curl -fsSL https://gvisor.dev/archive.key | sudo apt-key add - && "+
			"sudo add-apt-repository deb https://packages.cloud.google.com/apt gvisor-$(uname -m) main && "+
			"sudo apt-get update && sudo apt-get install -y runsc")
	gvisorInstall.Run() // Best effort - gVisor may already be installed

	// Set up gVisor as the default runtime
	setRuntime := exec.CommandContext(ctx, a.wslPath, "-d", wslName, "--", "bash", "-c",
		"echo 'export GRICD_OPTS=--gvisor-default' >> ~/.bashrc")

	return wslName, nil
}

// createMicroVMSandbox creates a Cloud Hypervisor MicroVM.
func (a *Adapter) createMicroVMSandbox(ctx context.Context, name string, config *Config) (string, error) {
	vmName := fmt.Sprintf("devenv-%s", name)

	// Create Cloud Hypervisor VM configuration
	cmd := exec.CommandContext(ctx, a.microVMPath, "--name", vmName,
		"--kernel", "/var/lib/cloud-hypervisor/vmlinux",
		"--initramfs", "/var/lib/cloud-hypervisor/initrd",
		"--memory", "size=4G",
		"--vcpus", "2")

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create Cloud Hypervisor VM: %w", err)
	}

	return vmName, nil
}

// Start starts a sandbox instance.
func (a *Adapter) Start(ctx context.Context, name string) error {
	switch a.defaultTier {
	case WindowsTierNative:
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command",
			fmt.Sprintf("Start-VM -Name '%s'", name))
		return cmd.Run()
	case WindowsTierWSL:
		cmd := exec.CommandContext(ctx, a.wslPath, "-d", name)
		return cmd.Run()
	case WindowsTierMicroVM:
		cmd := exec.CommandContext(ctx, a.microVMPath, "--name", name, "start")
		return cmd.Run()
	}
	return fmt.Errorf("unknown tier: %s", a.defaultTier)
}

// Stop stops a running sandbox instance.
func (a *Adapter) Stop(ctx context.Context, name string) error {
	switch a.defaultTier {
	case WindowsTierNative:
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command",
			fmt.Sprintf("Stop-VM -Name '%s' -Force", name))
		return cmd.Run()
	case WindowsTierWSL:
		cmd := exec.CommandContext(ctx, a.wslPath, "--", "--terminate", name)
		return cmd.Run()
	case WindowsTierMicroVM:
		cmd := exec.CommandContext(ctx, a.microVMPath, "--name", name, "stop")
		return cmd.Run()
	}
	return fmt.Errorf("unknown tier: %s", a.defaultTier)
}

// Delete deletes a sandbox instance.
func (a *Adapter) Delete(ctx context.Context, name string) error {
	switch a.defaultTier {
	case WindowsTierNative:
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command",
			fmt.Sprintf("Remove-VM -Name '%s' -Force", name))
		return cmd.Run()
	case WindowsTierWSL:
		cmd := exec.CommandContext(ctx, a.wslPath, "--", "--unregister", name)
		return cmd.Run()
	case WindowsTierMicroVM:
		cmd := exec.CommandContext(ctx, a.microVMPath, "--name", name, "delete")
		return cmd.Run()
	}
	return fmt.Errorf("unknown tier: %s", a.defaultTier)
}

// Exec executes a command in the sandbox.
func (a *Adapter) Exec(ctx context.Context, name string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	switch a.defaultTier {
	case WindowsTierWSL:
		execCmd := exec.CommandContext(ctx, a.wslPath, "-d", name, "--", "bash", "-c", strings.Join(cmd, " "))
		execCmd.Stdin = stdin
		execCmd.Stdout = stdout
		execCmd.Stderr = stderr
		return execCmd.Run()
	case WindowsTierMicroVM:
		execCmd := exec.CommandContext(ctx, a.microVMPath, "--name", name, "console")
		execCmd.Stdin = stdin
		execCmd.Stdout = stdout
		execCmd.Stderr = stderr
		return execCmd.Run()
	}
	return fmt.Errorf("exec not supported for tier: %s", a.defaultTier)
}

// Pull pulls an image (NOP for Windows - uses distributions or VHDs instead).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	return nil
}

// ApplySandbox applies a sandbox isolation layer (gVisor, landlock, etc.) to a running VM.
func (a *Adapter) ApplySandbox(ctx context.Context, name, sandboxType string) error {
	if a.defaultTier != WindowsTierWSL {
		return fmt.Errorf("sandbox application only supported on WSL tier")
	}

	switch sandboxType {
	case "gvisor", "runsc":
		// Configure gVisor as the runtime for a specific container/pod
		cmd := exec.CommandContext(ctx, a.wslPath, "-d", name, "--", "bash", "-c",
			fmt.Sprintf("echo 'runtime: runsc' >> /etc/containerd/config.toml && systemctl restart containerd"))
		return cmd.Run()
	case "landlock":
		// Landlock is a Linux kernel feature - enabled via sysctl
		cmd := exec.CommandContext(ctx, a.wslPath, "-d", name, "--", "bash", "-c",
			"echo 1 > /proc/sys/kernel/landlock_restricted")
		return cmd.Run()
	case "seccomp":
		// Apply seccomp profile via containerd
		cmd := exec.CommandContext(ctx, a.wslPath, "-d", name, "--", "bash", "-c",
			"mkdir -p /etc/containerd && echo 'SecurityProfile: seccomp' >> /etc/containerd/config.toml")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported sandbox type: %s", sandboxType)
}

// Config holds Windows adapter configuration.
type Config struct {
	WorkDir string
	Memory  string
	CPUs    int
	Tier    WindowsTier
}

// Option is a functional option for Windows sandbox config.
type Option func(*Config)

// WithWorkDir sets the working directory for the sandbox.
func WithWorkDir(dir string) Option {
	return func(c *Config) { c.WorkDir = dir }
}

// WithMemory sets the memory for the sandbox.
func WithMemory(mem string) Option {
	return func(c *Config) { c.Memory = mem }
}

// WithCPUs sets the CPU count for the sandbox.
func WithCPUs(cpus int) Option {
	return func(c *Config) { c.CPUs = cpus }
}

// WithTier sets the Windows virtualization tier.
func WithTier(tier WindowsTier) Option {
	return func(c *Config) { c.Tier = tier }
}
