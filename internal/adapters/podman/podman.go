// SPDX-License-Identifier: MIT OR Apache-2.0
// Package podman implements the daemon's SandboxPort using the Podman CLI.
//
// This adapter intentionally uses container IDs returned by `podman create`
// as the durable sandbox IDs.  Podman remains the source of truth for state;
// the adapter does not maintain a second lifecycle registry.
package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
)

// Adapter is a Podman-backed SandboxPort implementation.
type Adapter struct {
	binary string
}

// NewAdapter creates an adapter using podman from PATH, or the path supplied
// by NVMS_PODMAN_BINARY.  The environment override is useful for packaged
// installations and hermetic tests; it is not interpreted as a shell command.
func NewAdapter() *Adapter {
	binary := os.Getenv("NVMS_PODMAN_BINARY")
	if binary == "" {
		binary = "podman"
	}
	return &Adapter{binary: binary}
}

// NewAdapterWithBinary creates an adapter with an explicit executable path.
// It is primarily useful for tests and controlled host installations.
func NewAdapterWithBinary(binary string) *Adapter { return &Adapter{binary: binary} }

type inspected struct {
	ID      string `json:"Id"`
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State   struct {
		Status    string `json:"Status"`
		PID       int    `json:"Pid"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
		Env    []string          `json:"Env"`
	} `json:"Config"`
	NetworkSettings struct {
		IPAddress string `json:"IPAddress"`
	} `json:"NetworkSettings"`
}

func (a *Adapter) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, a.binary, args...)
}

func (a *Adapter) run(ctx context.Context, args ...string) ([]byte, error) {
	out, err := a.command(ctx, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("podman %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// Probe verifies that Podman is installed and responding without creating a
// container.
func (a *Adapter) Probe(ctx context.Context) error {
	if _, err := exec.LookPath(a.binary); err != nil {
		return fmt.Errorf("podman: binary %q not found: %w", a.binary, err)
	}
	if _, err := a.run(ctx, "version", "--format", "{{.Version}}"); err != nil {
		return fmt.Errorf("podman: runtime unavailable: %w", err)
	}
	return nil
}

// Create creates a stopped Podman container and returns its real container ID.
func (a *Adapter) Create(ctx context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Image) == "" {
		return nil, fmt.Errorf("podman: image is required")
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = "nvms-" + domain.GenerateID()
	}
	args := []string{"create", "--name", name}
	for key, value := range cfg.Environment {
		args = append(args, "--env", key+"="+value)
	}
	for key, value := range cfg.Labels {
		args = append(args, "--label", key+"="+value)
	}
	for _, mount := range cfg.Mounts {
		if mount.Source == "" || mount.Target == "" {
			return nil, fmt.Errorf("podman: mount source and target are required")
		}
		volume := mount.Source + ":" + mount.Target
		if mount.ReadOnly {
			volume += ":ro"
		}
		args = append(args, "--volume", volume)
	}
	if cfg.WorkDir != "" {
		args = append(args, "--workdir", cfg.WorkDir)
	}
	if cfg.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}
	if cfg.TmpfsTmp {
		args = append(args, "--tmpfs", "/tmp")
	}
	args = append(args, cfg.Image)
	if cfg.NativeSandbox != nil && len(cfg.NativeSandbox.Command) != 0 {
		args = append(args, cfg.NativeSandbox.Command...)
	}
	out, err := a.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		return nil, fmt.Errorf("podman create returned an empty container ID")
	}
	return a.inspect(ctx, id)
}

// Start starts a created container and refreshes its state from inspect.
func (a *Adapter) Start(ctx context.Context, id string) error {
	if _, err := a.run(ctx, "start", id); err != nil {
		return err
	}
	_, err := a.inspect(ctx, id)
	return err
}

// Stop stops a running container.
func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	verb := "stop"
	if force {
		verb = "kill"
	}
	_, err := a.run(ctx, verb, id)
	return err
}

// Delete removes a container and its metadata.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	_, err := a.run(ctx, "rm", "-f", id)
	return err
}

// List returns all containers known to Podman, including stopped containers.
func (a *Adapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	out, err := a.run(ctx, "ps", "-a", "--format", "{{.ID}}")
	if err != nil {
		return nil, err
	}
	var result []*domain.Sandbox
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		id := strings.TrimSpace(line)
		if id == "" {
			continue
		}
		sb, inspectErr := a.inspect(ctx, id)
		if inspectErr != nil {
			return nil, inspectErr
		}
		result = append(result, sb)
	}
	return result, nil
}

// Get returns the current Podman state for id.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	return a.inspect(ctx, id)
}

// Logs streams the container's stdout/stderr through Podman.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, id)
	cmd := a.command(ctx, args...)
	cmd.Stderr = cmd.Stdout
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("podman logs: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("podman logs: %w", err)
	}
	return &waitReadCloser{ReadCloser: r, wait: cmd.Wait}, nil
}

// Exec executes a command in a running container.
func (a *Adapter) Exec(ctx context.Context, id string, command []string) (io.ReadCloser, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("podman exec: command is required")
	}
	args := append([]string{"exec", id, "--"}, command...)
	cmd := a.command(ctx, args...)
	cmd.Stderr = cmd.Stdout
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("podman exec: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("podman exec: %w", err)
	}
	return &waitReadCloser{ReadCloser: r, wait: cmd.Wait}, nil
}

// Metrics returns a best-effort one-shot Podman stats sample.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	out, err := a.run(ctx, "stats", "--no-stream", "--format", "{{json .}}", id)
	if err != nil {
		return nil, err
	}
	var stats struct {
		CPU    string `json:"CPU %"`
		Memory string `json:"MemUsage"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &stats); err != nil {
		return nil, fmt.Errorf("podman stats: decode: %w", err)
	}
	m := &domain.SandboxMetrics{SandboxID: id}
	if cpu := strings.TrimSuffix(strings.TrimSpace(stats.CPU), "%"); cpu != "" {
		m.CPUUsage, _ = strconv.ParseFloat(cpu, 64)
	}
	if parts := strings.SplitN(stats.Memory, "/", 2); len(parts) == 2 {
		m.MemoryUsage = parseBytes(strings.TrimSpace(parts[0]))
	}
	return m, nil
}

type waitReadCloser struct {
	io.ReadCloser
	wait func() error
}

func (r *waitReadCloser) Close() error {
	closeErr := r.ReadCloser.Close()
	waitErr := r.wait()
	if closeErr != nil {
		return closeErr
	}
	return waitErr
}

func (a *Adapter) inspect(ctx context.Context, id string) (*domain.Sandbox, error) {
	out, err := a.run(ctx, "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return nil, err
	}
	var c inspected
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &c); err != nil {
		return nil, fmt.Errorf("podman inspect: decode: %w", err)
	}
	created := parseTime(c.Created)
	started := parseTime(c.State.StartedAt)
	var startedAt *time.Time
	if !started.IsZero() {
		startedAt = &started
	}
	status := domain.SandboxStatusStopped
	switch strings.ToLower(c.State.Status) {
	case "created", "configured", "initialized":
		status = domain.SandboxStatusPending
	case "running":
		status = domain.SandboxStatusRunning
	case "paused", "stopped", "exited", "dead", "removing":
		status = domain.SandboxStatusStopped
	}
	cfg := &domain.SandboxConfig{Name: strings.TrimPrefix(c.Name, "/"), Image: c.Config.Image}
	return &domain.Sandbox{
		ID: c.ID, Name: strings.TrimPrefix(c.Name, "/"), Status: status,
		Type: domain.SandboxTypeContainer, Config: cfg, PID: c.State.PID,
		CreatedAt: created, StartedAt: startedAt, IPAddress: c.NetworkSettings.IPAddress,
	}, nil
}

func parseTime(value string) time.Time {
	if value == "" || value == "0001-01-01T00:00:00Z" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05 -0700 MST"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseBytes(value string) int64 {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	multiplier := float64(1)
	if len(fields) > 1 {
		switch strings.ToUpper(fields[1]) {
		case "KB", "KIB":
			multiplier = 1024
		case "MB", "MIB":
			multiplier = 1024 * 1024
		case "GB", "GIB":
			multiplier = 1024 * 1024 * 1024
		}
	}
	return int64(n * multiplier)
}
