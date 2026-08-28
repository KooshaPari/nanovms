// Package firecracker implements ports.SandboxPort using Firecracker microVMs.
package firecracker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"github.com/kooshapari/nanovms/internal/domain"
)

type Adapter struct {
	mu   sync.Mutex
	vms  map[string]*domain.Sandbox
	cmds map[string]*exec.Cmd
}

func NewAdapter() *Adapter {
	return &Adapter{vms: make(map[string]*domain.Sandbox), cmds: make(map[string]*exec.Cmd)}
}

func (a *Adapter) Create(_ context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	sb := &domain.Sandbox{
		ID: "fc-" + domain.GenerateID(), Name: cfg.Name,
		Status: domain.SandboxStatusPending, Type: domain.SandboxTypeVM,
		VMFlavor: domain.VMFlavorMicroVM, Config: &cfg,
	}
	a.mu.Lock()
	a.vms[sb.ID] = sb
	a.mu.Unlock()
	return sb, nil
}

func (a *Adapter) Start(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.vms[id]; !ok {
		return fmt.Errorf("vm not found: %s", id)
	}
	bin, err := exec.LookPath("firecracker")
	if err != nil {
		return fmt.Errorf("firecracker not found: %w", err)
	}
	cmd := exec.CommandContext(ctx, bin, "--api-sock", "/tmp/fc-"+id+".sock", "--id", id)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("firecracker: %w", err)
	}
	a.cmds[id] = cmd
	a.vms[id].Status = domain.SandboxStatusRunning
	return nil
}

func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.vms[id]; !ok {
		return fmt.Errorf("vm not found: %s", id)
	}
	if cmd, ok := a.cmds[id]; ok && cmd.Process != nil {
		sig := "SIGTERM"
		if force {
			sig = "SIGKILL"
		}
		_ = exec.CommandContext(ctx, "kill", "-"+sig, fmt.Sprintf("%d", cmd.Process.Pid)).Run()
		_ = cmd.Wait()
	}
	_ = os.Remove("/tmp/fc-" + id + ".sock")
	a.vms[id].Status = domain.SandboxStatusStopped
	return nil
}

func (a *Adapter) Delete(ctx context.Context, id string) error {
	_ = a.Stop(ctx, id, true)
	a.mu.Lock()
	delete(a.vms, id)
	delete(a.cmds, id)
	a.mu.Unlock()
	return nil
}

func (a *Adapter) List(context.Context) ([]*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]*domain.Sandbox, 0, len(a.vms))
	for _, sb := range a.vms {
		out = append(out, sb)
	}
	return out, nil
}

func (a *Adapter) Get(_ context.Context, id string) (*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sb, ok := a.vms[id]
	if !ok {
		return nil, fmt.Errorf("vm not found: %s", id)
	}
	return sb, nil
}

func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	a.mu.Lock()
	cmd := a.cmds[id]
	a.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil, fmt.Errorf("vm not running: %s", id)
	}
	args := []string{"--no-pager", "-p", "info", "_PID=" + fmt.Sprintf("%d", cmd.Process.Pid)}
	if follow {
		args = append(args, "-f")
	}
	jcmd := exec.CommandContext(ctx, "journalctl", args...)
	r, err := jcmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	return r, jcmd.Start()
}

func (a *Adapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	a.mu.Lock()
	cmd := a.cmds[id]
	_, sbOK := a.vms[id]
	a.mu.Unlock()
	if !sbOK {
		return nil, fmt.Errorf("vm not found: %s", id)
	}
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil, fmt.Errorf("vm %s not running", id)
	}
	if _, err := exec.LookPath("jailer"); err != nil {
		return nil, fmt.Errorf("exec requires jailer; use firecracker vsock for VM-exec")
	}
	args := append([]string{"-t", fmt.Sprintf("%d", cmd.Process.Pid), "-m", "-u", "-i", "-p", "--"}, cmdArgs...)
	ncmd := exec.CommandContext(ctx, "nsenter", args...)
	r, err := ncmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	ncmd.Stderr = ncmd.Stdout
	return r, ncmd.Start()
}

func (a *Adapter) Metrics(_ context.Context, id string) (*domain.SandboxMetrics, error) {
	a.mu.Lock()
	sb, ok := a.vms[id]
	cmd := a.cmds[id]
	a.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("vm not found: %s", id)
	}
	m := &domain.SandboxMetrics{SandboxID: id}
	if cmd != nil && cmd.Process != nil && cmd.Process.Pid > 0 {
		if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", cmd.Process.Pid)); err == nil {
			var memKB int64
			_, _ = fmt.Sscanf(string(b), "VmRSS: %d kB", &memKB)
			m.MemoryUsage = memKB * 1024
		}
	}
	_ = sb
	return m, nil
}

func (a *Adapter) PortForward(ctx context.Context, id string, localPort, remotePort int) (string, error) {
	a.mu.Lock()
	if _, ok := a.vms[id]; !ok {
		a.mu.Unlock()
		return "", fmt.Errorf("vm not found: %s", id)
	}
	a.mu.Unlock()
	addr := fmt.Sprintf("127.0.0.1:%d", localPort)
	cmd := exec.CommandContext(ctx, "socat",
		fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", localPort),
		fmt.Sprintf("TCP:172.16.0.2:%d", remotePort))
	return addr, cmd.Start()
}
