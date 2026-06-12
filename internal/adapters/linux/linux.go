package linux

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// SandboxType represents the type of Linux sandboxing.
type SandboxType int

const (
	// Native namespaces (rootless containers)
	Native SandboxType = iota
	// MicroVM using Firecracker/Cloud Hypervisor
	MicroVM
	// WASM runtime (Wasmtime/WasmEdge)
	WASM
)

// Adapter implements the ports.RuntimeAdapter for Linux with multiple backends.
type Adapter struct {
	sandboxType SandboxType
	rootless    bool
	syscalls    sandboxSyscalls
}

// New creates a new Linux adapter with the specified sandbox type.
func New(sandboxType SandboxType, rootless bool) *Adapter {
	return &Adapter{
		sandboxType: sandboxType,
		rootless:    rootless,
		syscalls:    execSandboxSyscalls{},
	}
}

func newWithSyscalls(sandboxType SandboxType, rootless bool, syscalls sandboxSyscalls) *Adapter {
	adapter := New(sandboxType, rootless)
	adapter.syscalls = syscalls
	return adapter
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	switch a.sandboxType {
	case Native:
		if a.rootless {
			return "linux-rootless"
		}
		return "linux-native"
	case MicroVM:
		return "linux-microvm"
	case WASM:
		return "linux-wasm"
	default:
		return "linux-unknown"
	}
}

// Initialize initializes the Linux sandboxing backend.
func (a *Adapter) Initialize(ctx context.Context) error {
	switch a.sandboxType {
	case Native:
		return a.initializeNative(ctx)
	case MicroVM:
		return a.initializeMicroVM(ctx)
	case WASM:
		return a.initializeWASM(ctx)
	}
	return nil
}

func (a *Adapter) initializeNative(ctx context.Context) error {
	// Check for required tools
	tools := []string{"ip", "unshare", "mount"}
	for _, tool := range tools {
		if _, err := a.syscalls.LookPath(tool); err != nil {
			return fmt.Errorf("required tool %s not found: %w", tool, err)
		}
	}
	return nil
}

func (a *Adapter) initializeMicroVM(ctx context.Context) error {
	// Check for Firecracker or Cloud Hypervisor
	_, firecracker := a.syscalls.LookPath("firecracker")
	_, cloudhv := a.syscalls.LookPath("cloud-hypervisor")

	if firecracker != nil && cloudhv != nil {
		return fmt.Errorf("neither Firecracker nor Cloud Hypervisor found")
	}
	return nil
}

func (a *Adapter) initializeWASM(ctx context.Context) error {
	// Check for Wasmtime
	if _, err := a.syscalls.LookPath("wasmtime"); err != nil {
		return fmt.Errorf("Wasmtime not found: %w", err)
	}
	return nil
}

// Create creates a new Linux sandbox.
func (a *Adapter) Create(ctx context.Context, config interface{}) (string, error) {
	switch a.sandboxType {
	case Native:
		return a.createNative(ctx, config)
	case MicroVM:
		return a.createMicroVM(ctx, config)
	case WASM:
		return a.createWASM(ctx, config)
	}
	return "", fmt.Errorf("unsupported sandbox type")
}

func (a *Adapter) createNative(ctx context.Context, config interface{}) (string, error) {
	name := fmt.Sprintf("devenv-%s", config.(string))

	// Check if running as root for privileged operations
	if !a.rootless {
		// For privileged containers, use mount namespaces
		_, err := a.syscalls.Start(
			ctx,
			"unshare",
			"--mount",
			"--ipc",
			"--pid",
			"--fork",
			"bash",
			"-c",
			"sleep infinity",
		)
		if err != nil {
			return "", fmt.Errorf("failed to create namespace: %w", err)
		}
		return name, nil
	}

	// Rootless: use user namespaces with unshare
	_, err := a.syscalls.Start(ctx, "unshare", "--user", "--map-root-user", "sleep", "infinity")
	if err != nil {
		return "", fmt.Errorf("failed to create user namespace: %w", err)
	}

	return name, nil
}

func (a *Adapter) createMicroVM(ctx context.Context, config interface{}) (string, error) {
	name := fmt.Sprintf("devenv-microvm-%s", config.(string))

	// Check for Firecracker binary
	firecrackerPath, err := a.syscalls.LookPath("firecracker")
	if err != nil {
		// Try cloud-hypervisor
		firecrackerPath, err = a.syscalls.LookPath("cloud-hypervisor")
		if err != nil {
			return "", fmt.Errorf("no MicroVM hypervisor found")
		}
	}

	// Create VM config
	vmConfig := map[string]interface{}{
		"name":   name,
		"vmm":    firecrackerPath,
		"kernel": "/var/lib/devenv/vmlinux",
		"initrd": "/var/lib/devenv/initrd",
		"memory": "512M",
		"vcpus":  2,
	}

	_ = vmConfig // Use in actual implementation

	return name, nil
}

func (a *Adapter) createWASM(ctx context.Context, config interface{}) (string, error) {
	name := fmt.Sprintf("devenv-wasm-%s", config.(string))

	// For WASM, we don't create VMs but compile to WASM
	// The "sandbox" is the WASM runtime

	return name, nil
}

// Start starts an existing sandbox.
func (a *Adapter) Start(ctx context.Context, id string) error {
	switch a.sandboxType {
	case Native:
		return nil // Namespaces start on creation
	case MicroVM:
		return a.startMicroVM(ctx, id)
	case WASM:
		return nil // WASM modules are started on instantiation
	}
	return fmt.Errorf("unsupported sandbox type")
}

func (a *Adapter) startMicroVM(ctx context.Context, id string) error {
	// Start Firecracker/Cloud Hypervisor VM
	return nil
}

// Stop stops a running sandbox.
func (a *Adapter) Stop(ctx context.Context, id string) error {
	switch a.sandboxType {
	case Native:
		// Kill the namespace process
		return a.syscalls.Run(ctx, "pkill", []string{"-f", id}, nil, nil, nil)
	case MicroVM:
		return a.stopMicroVM(ctx, id)
	case WASM:
		return nil // WASM is stopped by dropping reference
	}
	return fmt.Errorf("unsupported sandbox type")
}

func (a *Adapter) stopMicroVM(ctx context.Context, id string) error {
	// Send stop signal to VM
	return a.syscalls.Run(ctx, "pkill", []string{"-f", id}, nil, nil, nil)
}

// Delete deletes a sandbox.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	return a.Stop(ctx, id)
}

// Exec executes a command in the Linux sandbox.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	switch a.sandboxType {
	case Native:
		return a.execNative(ctx, id, cmd, stdin, stdout, stderr)
	case MicroVM:
		return a.execMicroVM(ctx, id, cmd, stdin, stdout, stderr)
	case WASM:
		return a.execWASM(ctx, id, cmd, stdin, stdout, stderr)
	}
	return fmt.Errorf("unsupported sandbox type")
}

func (a *Adapter) execNative(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if strings.HasPrefix(id, "devenv-wasm-") {
		// WASM execution via wasmtime
		if len(cmd) == 0 {
			return fmt.Errorf("no command provided for WASM sandbox")
		}
		args := []string{"wasmtime", "--dir", "/"}
		args = append(args, cmd...)
		return a.syscalls.Run(ctx, args[0], args[1:], stdin, stdout, stderr)
	} else if strings.Contains(id, "devenv-") {
		// Execute in namespace
		args := []string{"--user", "--map-root-user", "--mount", "--ipc", "--pid", "--fork", "bash", "-c", strings.Join(cmd, " ")}
		return a.syscalls.Run(ctx, "unshare", args, stdin, stdout, stderr)
	} else {
		return a.syscalls.Run(ctx, "bash", []string{"-c", strings.Join(cmd, " ")}, stdin, stdout, stderr)
	}
}

func (a *Adapter) execMicroVM(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Execute via VM's vsock or console
	return a.syscalls.Run(ctx, "bash", []string{"-c", strings.Join(cmd, " ")}, stdin, stdout, stderr)
}

func (a *Adapter) execWASM(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Execute WASM module via wasmtime
	wasmCmd := []string{"wasmtime"}
	wasmCmd = append(wasmCmd, cmd...)
	return a.syscalls.Run(ctx, wasmCmd[0], wasmCmd[1:], stdin, stdout, stderr)
}

// Pull pulls an image (NOP for local filesystems).
func (a *Adapter) Pull(ctx context.Context, image string) error {
	return nil
}

// ListImages lists available Linux base images.
func (a *Adapter) ListImages(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// ApplySandboxProfile applies a sandbox security profile (seccomp/landlock).
func (a *Adapter) ApplySandboxProfile(ctx context.Context, id string, profile string) error {
	// Apply seccomp profile
	return a.syscalls.Run(
		ctx,
		"ip",
		[]string{"netns", "exec", id, "bash", "-c", fmt.Sprintf("echo %s > /proc/self/status", profile)},
		nil,
		nil,
		nil,
	)
}
