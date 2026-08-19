// SPDX-License-Identifier: MIT OR Apache-2.0
package gpu

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandResult contains bounded command output.
type CommandResult struct {
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Duration  time.Duration
	TimedOut  bool
	Truncated bool
	// Hashes cover the complete captured (and therefore bounded) byte slices.
	StdoutSHA256 [sha256.Size]byte
	StderrSHA256 [sha256.Size]byte
}

// CommandRunner makes inventory adapters deterministic and injectable.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

// ExecRunner runs commands with a mandatory timeout and bounded output.
type ExecRunner struct {
	Timeout   time.Duration
	MaxOutput int
}

// Run executes one command without a shell.
func (runner ExecRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	return runner.run(ctx, nil, name, args...)
}

// RunWithEnv executes one command with explicit environment overrides. It is
// used by bounded host actions that must select a particular local transport.
func (runner ExecRunner) RunWithEnv(ctx context.Context, environment map[string]string, name string, args ...string) (CommandResult, error) {
	return runner.run(ctx, environment, name, args...)
}

func (runner ExecRunner) run(ctx context.Context, environment map[string]string, name string, args ...string) (CommandResult, error) {
	if runner.Timeout <= 0 {
		return CommandResult{}, fmt.Errorf("command timeout must be positive")
	}
	if runner.MaxOutput <= 0 {
		return CommandResult{}, fmt.Errorf("command output bound must be positive")
	}
	runCtx, cancel := context.WithTimeout(ctx, runner.Timeout)
	defer cancel()

	var stdout, stderr boundedBuffer
	stdout.limit = runner.MaxOutput
	stderr.limit = runner.MaxOutput
	started := time.Now()
	command := exec.CommandContext(runCtx, name, args...)
	command.Env = mergeEnvironment(os.Environ(), environment)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	// Take ownership of the bounded buffers' backing arrays; they are not
	// reused after this return, so Clone would only double allocate.
	result := CommandResult{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.Bytes(),
		ExitCode:  exitCode(err),
		Duration:  time.Since(started),
		TimedOut:  errors.Is(runCtx.Err(), context.DeadlineExceeded),
		Truncated: stdout.overflow || stderr.overflow,
	}
	result.StdoutSHA256 = sha256.Sum256(result.Stdout)
	result.StderrSHA256 = sha256.Sum256(result.Stderr)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return result, fmt.Errorf("command %q timed out after %s: %w", name, runner.Timeout, context.DeadlineExceeded)
	}
	if stdout.overflow || stderr.overflow {
		return result, fmt.Errorf("command %q exceeded %d-byte output bound", name, runner.MaxOutput)
	}
	if err != nil {
		return result, fmt.Errorf("command %q failed: %w", name, err)
	}
	return result, nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	result := make([]string, len(base), len(base)+len(overrides))
	copy(result, base)
	for key, value := range overrides {
		replaced := false
		for i, item := range result {
			existing, _, _ := strings.Cut(item, "=")
			if strings.EqualFold(existing, key) {
				result[i] = key + "=" + value
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, key+"="+value)
		}
	}
	return result
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.overflow = true
		return originalLength, nil
	}
	if len(data) > remaining {
		buffer.overflow = true
		data = data[:remaining]
	}
	_, _ = buffer.Buffer.Write(data)
	return originalLength, nil
}
