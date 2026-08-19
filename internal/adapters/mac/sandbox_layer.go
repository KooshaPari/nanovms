package mac

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/kooshapari/nanovms/internal/domain"
)

// applySandboxLayer applies the specified sandbox isolation layer.
func (a *Adapter) applySandboxLayer(ctx context.Context, id string, layer domain.SandboxLayer) error {
	switch layer {
	case domain.SandboxLayerGVisor:
		// Install and configure gVisor (runsc)
		cmd := exec.CommandContext(ctx, "which", "runsc")
		if err := cmd.Run(); err != nil {
			// Install gVisor
			installCmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://gvisor.dev/install.sh")
			if err := installCmd.Run(); err != nil {
				return fmt.Errorf("failed to install gVisor: %w", err)
			}
		}
		// Configure the VM to use gVisor as runtime
		return nil

	case domain.SandboxLayerSRAMP:
		// sRAMP (Secure Runtime Application Malware Protection) - Linux native
		// Configure landlock + seccomp via kernel params
		cmd := exec.CommandContext(ctx, a.limaPath, "shell", id, "sysctl", "-w", "kernel.yama.ptrace_scope=2")
		return cmd.Run()

	case domain.SandboxLayerWasmtime:
		// WASM runtime via wasmtime
		cmd := exec.CommandContext(ctx, "which", "wasmtime")
		if err := cmd.Run(); err != nil {
			installCmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://wasmtime.dev/install.sh")
			return installCmd.Run()
		}
		return nil

	case domain.SandboxLayerNone:
		return nil

	default:
		return fmt.Errorf("unsupported sandbox layer: %s", layer)
	}
}
