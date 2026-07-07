// Package sandbox / helpers.go — utility functions (logs, exec, metrics, ID generation).
//
// Part of the nanovms sandbox adapter decomposition (R-A P3).

// Package sandbox provides the sandbox isolation layer adapter.
// It implements the SandboxPort interface for various sandboxing technologies
// including gVisor, landlock, seccomp, and wasmtime.
package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"github.com/kooshapari/nanovms/internal/domain"
)

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
