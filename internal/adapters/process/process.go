// Package process provides a concrete process-based sandbox adapter.
package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// Adapter implements ports.SandboxPort using native OS process execution.
// It tracks sandbox metadata and real process lifecycle (PID, stop, exec, delete).
type Adapter struct {
	mu        sync.Mutex
	sandboxes map[string]*processSandbox
}

type processSandbox struct {
	sandbox  *domain.Sandbox
	cmd      *exec.Cmd
	logPath  string
	waitDone chan struct{}
}

// NewAdapter creates a new process adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		sandboxes: make(map[string]*processSandbox),
	}
}

// Create creates a sandbox entry and assigns a sandbox ID.
func (a *Adapter) Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error) {
	id := domain.GenerateID()
	now := time.Now()
	sb := &domain.Sandbox{
		ID:          id,
		Name:        config.Name,
		Status:      domain.SandboxStatusPending,
		Type:        domain.SandboxTypeProcess,
		Config:      &config,
		PID:         -1,
		CreatedAt:   now,
		Mounts:      config.Mounts,
		Environment: config.Environment,
	}
	if sb.Name == "" {
		sb.Name = id
	}

	a.mu.Lock()
	a.sandboxes[id] = &processSandbox{sandbox: sb}
	a.mu.Unlock()
	return sb, nil
}

// Start launches the sandbox command (or default cross-platform command) and
// transitions the sandbox to running state.
func (a *Adapter) Start(ctx context.Context, id string) error {
	a.mu.Lock()
	entry, exists := a.sandboxes[id]
	if !exists {
		a.mu.Unlock()
		return fmt.Errorf("sandbox not found: %s", id)
	}
	if entry.sandbox.Status == domain.SandboxStatusRunning {
		a.mu.Unlock()
		return fmt.Errorf("sandbox already running: %s", id)
	}

	command := resolveStartCommand(entry.sandbox.Config)
	logFile, err := os.CreateTemp("", "nanovms-process-*.log")
	if err != nil {
		a.mu.Unlock()
		return fmt.Errorf("failed to allocate process log file: %w", err)
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = mergeEnvironment(entry.sandbox.Environment)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = os.Remove(logFile.Name())
		a.mu.Unlock()
		return fmt.Errorf("failed to start sandbox process: %w", err)
	}

	entry.cmd = cmd
	entry.waitDone = make(chan struct{})
	entry.logPath = logFile.Name()
	entry.sandbox.PID = cmd.Process.Pid
	now := time.Now()
	entry.sandbox.StartedAt = &now
	entry.sandbox.Status = domain.SandboxStatusRunning
	a.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
		close(entry.waitDone)
		a.mu.Lock()
		if entry, ok := a.sandboxes[id]; ok && entry.sandbox.Status == domain.SandboxStatusRunning {
			entry.sandbox.Status = domain.SandboxStatusStopped
			entry.sandbox.PID = -1
		}
		a.mu.Unlock()
	}()

	return nil
}

// Stop terminates the running sandbox process.
func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	a.mu.Lock()
	entry, exists := a.sandboxes[id]
	if !exists {
		a.mu.Unlock()
		return fmt.Errorf("sandbox not found: %s", id)
	}
	cmd := entry.cmd
	if entry.sandbox.Status != domain.SandboxStatusRunning || cmd == nil || cmd.Process == nil {
		entry.sandbox.Status = domain.SandboxStatusStopped
		entry.sandbox.PID = -1
		entry.cmd = nil
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	var stopErr error
	if runtime.GOOS == "windows" {
		stopErr = cmd.Process.Kill()
	} else {
		sig := syscall.SIGTERM
		if force {
			sig = syscall.SIGKILL
		}
		stopErr = cmd.Process.Signal(sig)
		if stopErr != nil && force {
			stopErr = cmd.Process.Kill()
		}
	}

	// The reaper goroutine owns cmd.Wait. Waiting here for its completion avoids
	// concurrent Wait calls, which race in os/exec under -race.
	<-entry.waitDone

	a.mu.Lock()
	entry.sandbox.Status = domain.SandboxStatusStopped
	entry.sandbox.PID = -1
	entry.cmd = nil
	a.mu.Unlock()
	return stopErr
}

// Delete tears down process state and removes sandbox metadata.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	a.mu.Lock()
	entry, exists := a.sandboxes[id]
	a.mu.Unlock()
	if !exists {
		return nil
	}

	if entry.sandbox.Status == domain.SandboxStatusRunning {
		_ = a.Stop(ctx, id, true)
	}

	a.mu.Lock()
	delete(a.sandboxes, id)
	logPath := entry.logPath
	a.mu.Unlock()

	if logPath != "" {
		_ = os.Remove(logPath)
	}
	return nil
}

// List returns all known sandboxes.
func (a *Adapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, entry := range a.sandboxes {
		result = append(result, entry.sandbox)
	}
	return result, nil
}

// Get returns a sandbox by ID.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return entry.sandbox, nil
}

// Logs returns the sandbox process logs.
//
// Follow mode returns the current log snapshot for portability and is safe for CI.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	a.mu.Lock()
	entry, exists := a.sandboxes[id]
	if !exists {
		a.mu.Unlock()
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	logPath := entry.logPath
	a.mu.Unlock()
	if logPath == "" {
		return nil, fmt.Errorf("sandbox logs unavailable for id=%s", id)
	}
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sandbox log: %w", err)
	}
	_ = follow
	return file, nil
}

// Exec executes a command for a running sandbox and returns its stdout stream.
func (a *Adapter) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	a.mu.Lock()
	entry, exists := a.sandboxes[id]
	if !exists {
		a.mu.Unlock()
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	sandbox := entry.sandbox
	a.mu.Unlock()

	if sandbox.Status != domain.SandboxStatusRunning {
		return nil, fmt.Errorf("sandbox not running: %s", id)
	}
	if len(cmd) == 0 {
		return nil, fmt.Errorf("no command provided")
	}

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Env = mergeEnvironment(sandbox.Environment)
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	execCmd.Stderr = execCmd.Stdout
	if err := execCmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to start exec command: %w", err)
	}
	go func() {
		_ = execCmd.Wait()
	}()
	return stdout, nil
}

// Metrics returns sandbox-level metrics for compatibility.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, exists := a.sandboxes[id]
	if !exists {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return &domain.SandboxMetrics{SandboxID: entry.sandbox.ID}, nil
}

func resolveStartCommand(config *domain.SandboxConfig) []string {
	if config != nil && config.NativeSandbox != nil && len(config.NativeSandbox.Command) > 0 {
		return config.NativeSandbox.Command
	}
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", "timeout /T 3600 /NOBREAK >NUL"}
	}
	return []string{"/bin/sh", "-c", "sleep 3600"}
}

func mergeEnvironment(env map[string]string) []string {
	// Safe environment variable allowlist — prevents leaking secrets,
	// LD_PRELOAD, and other sensitive host variables into sandboxes.
	safeVars := []string{
		"HOME", "USER", "SHELL", "LANG", "LC_ALL", "LC_CTYPE",
		"PATH", "TMPDIR", "TMP", "TEMP",
		"TERM", "COLORTERM", "NO_COLOR",
		"USERPROFILE", "APPDATA", "LOCALAPPDATA", // Windows
		"SystemRoot", "windir", "ProgramFiles",     // Windows
	}

	merged := make([]string, 0, len(safeVars)+len(env))
	for _, key := range safeVars {
		if val, ok := os.LookupEnv(key); ok {
			merged = append(merged, key+"="+val)
		}
	}
	for key, value := range env {
		merged = append(merged, key+"="+value)
	}
	return merged
}

var _ ports.SandboxPort = (*Adapter)(nil)
