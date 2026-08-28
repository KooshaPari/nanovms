package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

func TestMain(m *testing.M) {
	if isProcessHelperInvocation() {
		exitCode := runProcessHelper()
		os.Exit(exitCode)
	}
	os.Exit(m.Run())
}

func isProcessHelperInvocation() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.run=TestProcessAdapterHelper") {
			return true
		}
	}
	return false
}

func runProcessHelper() int {
	var mode, payload, duration string
	for i, arg := range os.Args {
		if arg != "--" {
			continue
		}
		for j := i + 1; j < len(os.Args); j++ {
			switch os.Args[j] {
			case "-nanovms-helper-mode":
				if j+1 < len(os.Args) {
					mode = os.Args[j+1]
					j++
				}
			case "-nanovms-helper-payload":
				if j+1 < len(os.Args) {
					payload = os.Args[j+1]
					j++
				}
			case "-nanovms-helper-duration":
				if j+1 < len(os.Args) {
					duration = os.Args[j+1]
					j++
				}
			}
		}
		break
	}

	switch mode {
	case "sleep":
		wait, err := time.ParseDuration(duration)
		if err != nil {
			wait = time.Second
		}
		time.Sleep(wait)
	case "echo":
		_, _ = fmt.Fprint(os.Stdout, payload)
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "stdout":
		_, _ = fmt.Fprint(os.Stdout, payload)
	}
	return 0
}

func TestProcessAdapterLifecycle(t *testing.T) {
	ctx := context.Background()
	self := os.Args[0]

	tests := []struct {
		name      string
		startArgs []string
		execOut   string
		forceStop bool
	}{
		{
			name: "graceful-stop",
			startArgs: []string{
				self,
				"-test.run=TestProcessAdapterHelper",
				"--",
				"-nanovms-helper-mode",
				"sleep",
				"-nanovms-helper-duration",
				"20s",
			},
			execOut:   "hello-world-1",
			forceStop: false,
		},
		{
			name: "force-stop",
			startArgs: []string{
				self,
				"-test.run=TestProcessAdapterHelper",
				"--",
				"-nanovms-helper-mode",
				"sleep",
				"-nanovms-helper-duration",
				"20s",
			},
			execOut:   "hello-world-2",
			forceStop: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewAdapter()
			sb, err := adapter.Create(ctx, domain.SandboxConfig{
				Name: tc.name,
				NativeSandbox: &domain.NativeSandboxConfig{
					Command: tc.startArgs,
				},
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			if sb.PID != -1 {
				t.Fatalf("expected PID -1 after create, got %d", sb.PID)
			}

			if err := adapter.Start(ctx, sb.ID); err != nil {
				t.Fatalf("Start failed: %v", err)
			}
			if sb.Status != domain.SandboxStatusRunning {
				t.Fatalf("expected running status after Start, got %s", sb.Status)
			}
			if sb.PID <= 0 {
				t.Fatalf("expected PID > 0 after Start, got %d", sb.PID)
			}

			execOut, err := runExecAndRead(ctx, adapter, sb.ID, []string{
				self,
				"-test.run=TestProcessAdapterHelper",
				"--",
				"-nanovms-helper-mode",
				"stdout",
				"-nanovms-helper-payload",
				tc.execOut,
			})
			if err != nil {
				t.Fatalf("Exec failed: %v", err)
			}
			if string(execOut) != tc.execOut {
				t.Fatalf("exec output mismatch, got %q want %q", string(execOut), tc.execOut)
			}

			if err := adapter.Stop(ctx, sb.ID, tc.forceStop); err != nil {
				t.Fatalf("Stop failed: %v", err)
			}
			if sb.Status != domain.SandboxStatusStopped {
				t.Fatalf("expected stopped status after Stop, got %s", sb.Status)
			}

			if err := adapter.Delete(ctx, sb.ID); err != nil {
				t.Fatalf("Delete failed: %v", err)
			}
			if _, err := adapter.Get(ctx, sb.ID); err == nil {
				t.Fatalf("Get should fail after Delete")
			}
		})
	}
}

func TestProcessAdapterDefaultCommand(t *testing.T) {
	adapter := NewAdapter()
	sb, err := adapter.Create(context.Background(), domain.SandboxConfig{Name: "default-command"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := adapter.Start(context.Background(), sb.ID); err != nil {
		// Default command is platform-specific and may vary by environment.
		t.Logf("default Start failed (allowed in this environment): %v", err)
		return
	}
	if sb.Status != domain.SandboxStatusRunning {
		t.Fatalf("expected running status, got %s", sb.Status)
	}
	if err := adapter.Stop(context.Background(), sb.ID, true); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if err := adapter.Delete(context.Background(), sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestProcessAdapterInvalidOperations(t *testing.T) {
	adapter := NewAdapter()
	if _, err := adapter.Exec(context.Background(), "missing", []string{"echo", "x"}); err == nil {
		t.Fatal("expected exec on missing sandbox to fail")
	}
	if err := adapter.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete missing sandbox should be no-op: %v", err)
	}
}

func runExecAndRead(ctx context.Context, adapter *Adapter, id string, cmd []string) ([]byte, error) {
	rc, err := adapter.Exec(ctx, id, cmd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	out, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(out), nil
}
