// Package gvisor implements ports.SandboxPort using gVisor (runsc).
package gvisor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"github.com/kooshapari/nanovms/internal/domain"
)

type Adapter struct {
	mu        sync.Mutex
	sandboxes map[string]*domain.Sandbox
	cmds      map[string]*exec.Cmd
}

func NewAdapter() *Adapter {
	return &Adapter{sandboxes: make(map[string]*domain.Sandbox), cmds: make(map[string]*exec.Cmd)}
}

func (a *Adapter) Create(_ context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	sb := &domain.Sandbox{
		ID: "gv-" + domain.GenerateID(), Name: cfg.Name,
		Status: domain.SandboxStatusPending, Type: domain.SandboxTypeGVisor, Config: &cfg,
	}
	a.mu.Lock()
	a.sandboxes[sb.ID] = sb
	a.mu.Unlock()
	return sb, nil
}

func (a *Adapter) Start(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sandboxes[id]; !ok {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	bin, err := exec.LookPath("runsc")
	if err != nil {
		return fmt.Errorf("runsc not found: %w", err)
	}
	if err := exec.CommandContext(ctx, bin, "run", "-detach", id).Run(); err != nil {
		return fmt.Errorf("runsc run: %w", err)
	}
	wcmd := exec.CommandContext(ctx, bin, "wait", id)
	_ = wcmd.Start()
	a.cmds[id] = wcmd
	a.sandboxes[id].Status = domain.SandboxStatusRunning
	return nil
}

func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sandboxes[id]; !ok {
		return fmt.Errorf("sandbox not found: %s", id)
	}
	sig := "SIGTERM"
	if force {
		sig = "SIGKILL"
	}
	bin, _ := exec.LookPath("runsc")
	return exec.CommandContext(ctx, bin, "kill", "-"+sig, id).Run()
}

func (a *Adapter) Delete(ctx context.Context, id string) error {
	_ = a.Stop(ctx, id, true)
	a.mu.Lock()
	delete(a.sandboxes, id)
	delete(a.cmds, id)
	a.mu.Unlock()
	return nil
}

func (a *Adapter) List(context.Context) ([]*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]*domain.Sandbox, 0, len(a.sandboxes))
	for _, sb := range a.sandboxes {
		out = append(out, sb)
	}
	return out, nil
}

func (a *Adapter) Get(_ context.Context, id string) (*domain.Sandbox, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sb, ok := a.sandboxes[id]
	if !ok {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	return sb, nil
}

func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	a.mu.Lock()
	if _, ok := a.sandboxes[id]; !ok {
		a.mu.Unlock()
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	a.mu.Unlock()
	bin, _ := exec.LookPath("runsc")
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, bin, append(args, id)...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	return r, cmd.Start()
}

func (a *Adapter) Exec(ctx context.Context, id string, cmdArgs []string) (io.ReadCloser, error) {
	a.mu.Lock()
	if _, ok := a.sandboxes[id]; !ok {
		a.mu.Unlock()
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	a.mu.Unlock()
	bin, _ := exec.LookPath("runsc")
	cmd := exec.CommandContext(ctx, bin, append([]string{"exec", id, "--"}, cmdArgs...)...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	return r, cmd.Start()
}

func (a *Adapter) Metrics(_ context.Context, id string) (*domain.SandboxMetrics, error) {
	a.mu.Lock()
	if _, ok := a.sandboxes[id]; !ok {
		a.mu.Unlock()
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	a.mu.Unlock()
	m := &domain.SandboxMetrics{SandboxID: id}
	bin, _ := exec.LookPath("runsc")
	if out, err := exec.Command(bin, "events", "--stats", id).Output(); err == nil {
		_, _ = fmt.Sscanf(string(out), "%f", &m.CPUUsage)
	}
	return m, nil
}

func (a *Adapter) PortForward(ctx context.Context, id string, localPort, remotePort int) (string, error) {
	a.mu.Lock()
	if _, ok := a.sandboxes[id]; !ok {
		a.mu.Unlock()
		return "", fmt.Errorf("sandbox not found: %s", id)
	}
	a.mu.Unlock()
	addr := fmt.Sprintf("127.0.0.1:%d", localPort)
	cmd := exec.CommandContext(ctx, "socat",
		fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", localPort),
		fmt.Sprintf("TCP:172.16.0.2:%d", remotePort))
	return addr, cmd.Start()
}
