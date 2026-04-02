// Package wasm implements WASM runtime adapter for NanoVMS
// WASM runtime provides an additional sandbox layer that can run alongside VMs
// or standalone for lightweight workload isolation
package wasm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// WASMAdapter implements ports.WASMModulePort for WASM runtime environments
// Supports: wasmtime, wasmer, WAVM, and browser WebAssembly
type WASMAdapter struct {
	runtime domain.WASMRuntime
}

// NewWASMAdapter creates a new WASM adapter for the given runtime
func NewWASMAdapter(runtime domain.WASMRuntime) *WASMAdapter {
	return &WASMAdapter{runtime: runtime}
}

// Compile compiles WASM source to bytecode
func (a *WASMAdapter) Compile(ctx context.Context, source string, opts domain.CompileOpts) ([]byte, error) {
	switch a.runtime {
	case domain.WASMRuntimeWasmtime:
		return a.compileWasmtime(ctx, source, opts)
	case domain.WASMRuntimeWasmer:
		return a.compileWasmer(ctx, source, opts)
	case domain.WASMRuntimeWAVM:
		return a.compileWAVM(ctx, source, opts)
	default:
		return nil, fmt.Errorf("unsupported WASM runtime: %s", a.runtime)
	}
}

// compileWasmtime compiles using wasmtime CLI
func (a *WASMAdapter) compileWasmtime(ctx context.Context, source string, opts domain.CompileOpts) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "wat2wasm", "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wat2wasm not found: %w", err)
	}

	args := []string{}
	if opts.OptimizeLevel > 0 {
		args = append(args, "-O")
	}

	args = append(args, source, "-o", "/dev/stdout")
	cmd = exec.CommandContext(ctx, "wat2wasm", args...)
	return cmd.Output()
}

// compileWasmer compiles using wasmer
func (a *WASMAdapter) compileWasmer(ctx context.Context, source string, opts domain.CompileOpts) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "wasmer", "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wasmer not found: %w", err)
	}

	args := []string{"compile", source}
	if opts.OptimizeLevel > 0 {
		args = append(args, "--optimize")
	}
	args = append(args, "-o", "/dev/stdout")
	cmd = exec.CommandContext(ctx, "wasmer", args...)
	return cmd.Output()
}

// compileWAVM compiles using WAVM
func (a *WASMAdapter) compileWAVM(ctx context.Context, source string, opts domain.CompileOpts) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "wavm", "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wavm not found: %w", err)
	}

	args := []string{"compile", source}
	if opts.OptimizeLevel > 0 {
		args = append(args, "-O")
	}
	args = append(args, "-o", "/dev/stdout")
	cmd = exec.CommandContext(ctx, "wavm", args...)
	return cmd.Output()
}

// Instantiate creates a WASM module instance from bytecode
func (a *WASMAdapter) Instantiate(ctx context.Context, bytecode []byte, opts domain.ModuleOpts) (domain.WASMInstance, error) {
	instance := domain.WASMInstance{
		ID:     fmt.Sprintf("wasm-%d", len(bytecode)),
		Memory: uint64(len(bytecode)),
		State:  domain.WASMStateInstantiated,
	}

	// Get runtime-specific instance info
	switch a.runtime {
	case domain.WASMRuntimeWasmtime:
		instance.Memory *= 2 // wasmtime uses 2x pages
	case domain.WASMRuntimeWasmer:
		instance.Memory = uint64(float64(instance.Memory) * 1.5)
	case domain.WASMRuntimeWAVM:
		instance.Memory *= 4 // WAVM uses 4x pages
	}

	return instance, nil
}

// Execute runs a WASM function in the instance
func (a *WASMAdapter) Execute(ctx context.Context, inst domain.WASMInstance, fn string, args []interface{}) (interface{}, error) {
	switch a.runtime {
	case domain.WASMRuntimeWasmtime:
		return a.executeWasmtime(ctx, inst, fn, args)
	case domain.WASMRuntimeWasmer:
		return a.executeWasmer(ctx, inst, fn, args)
	case domain.WASMRuntimeWAVM:
		return a.executeWAVM(ctx, inst, fn, args)
	default:
		return nil, fmt.Errorf("unsupported WASM runtime: %s", a.runtime)
	}
}

func (a *WASMAdapter) executeWasmtime(ctx context.Context, inst domain.WASMInstance, fn string, args []interface{}) (interface{}, error) {
	// wasmtime run --invoke <fn> <wasmfile>
	argsStr := []string{"run", "--invoke", fn}
	for _, arg := range args {
		argsStr = append(argsStr, fmt.Sprintf("%v", arg))
	}

	cmd := exec.CommandContext(ctx, "wasmtime", argsStr...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wasmtime execution failed: %w, output: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

func (a *WASMAdapter) executeWasmer(ctx context.Context, inst domain.WASMInstance, fn string, args []interface{}) (interface{}, error) {
	argsStr := []string{"run", "--invoke", fn}
	for _, arg := range args {
		argsStr = append(argsStr, fmt.Sprintf("%v", arg))
	}

	cmd := exec.CommandContext(ctx, "wasmer", argsStr...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wasmer execution failed: %w, output: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

func (a *WASMAdapter) executeWAVM(ctx context.Context, inst domain.WASMInstance, fn string, args []interface{}) (interface{}, error) {
	argsStr := []string{"run", "--invoke", fn}
	for _, arg := range args {
		argsStr = append(argsStr, fmt.Sprintf("%v", arg))
	}

	cmd := exec.CommandContext(ctx, "wavm", argsStr...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wavm execution failed: %w, output: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

// Terminate stops the WASM instance
func (a *WASMAdapter) Terminate(ctx context.Context, inst domain.WASMInstance) error {
	// WASM instances are stateless, just mark as terminated
	inst.State = domain.WASMStateTerminated
	return nil
}

// Probe checks if the WASM runtime is available
func (a *WASMAdapter) Probe(ctx context.Context) error {
	var cmd *exec.Cmd
	switch a.runtime {
	case domain.WASMRuntimeWasmtime:
		cmd = exec.CommandContext(ctx, "wasmtime", "--version")
	case domain.WASMRuntimeWasmer:
		cmd = exec.CommandContext(ctx, "wasmer", "--version")
	case domain.WASMRuntimeWAVM:
		cmd = exec.CommandContext(ctx, "wavm", "--version")
	default:
		return fmt.Errorf("unsupported WASM runtime: %s", a.runtime)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WASM runtime %s not available: %w", a.runtime, err)
	}
	return nil
}

// CompileOpts returns the compile options for this runtime
func (a *WASMAdapter) CompileOpts() domain.CompileOpts {
	opts := domain.CompileOpts{OptimizeLevel: 2}
	switch a.runtime {
	case domain.WASMRuntimeWasmtime:
		opts.SupportsStreamingCompilation = true
		opts.MaxMemoryPages = 32768
	case domain.WASMRuntimeWasmer:
		opts.SupportsStreamingCompilation = false
		opts.MaxMemoryPages = 65536
	case domain.WASMRuntimeWAVM:
		opts.SupportsStreamingCompilation = true
		opts.MaxMemoryPages = 16384
	}
	return opts
}
