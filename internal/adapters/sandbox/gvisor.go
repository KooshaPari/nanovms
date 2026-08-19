// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- gVisor adapter using Docker with --runtime=runsc.

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// GVisorAdapter runs containers through Docker with the gVisor runsc runtime.
type GVisorAdapter struct {
	mu       sync.Mutex
	docker   string
	runtime  string
	registry map[SandboxID]*gvisorContainer
}

type gvisorContainer struct {
	id     SandboxID
	config SandboxConfig
	status SandboxStatus
}

// NewGVisorAdapter creates a gVisor adapter that delegates to Docker with --runtime=runsc.
func NewGVisorAdapter() *GVisorAdapter {
	client := "docker"
	if p, err := exec.LookPath("docker"); err == nil {
		client = p
	}
	runtime := "runsc"
	if p, err := exec.LookPath("runsc"); err == nil {
		runtime = p
	}
	return &GVisorAdapter{
		docker:   client,
		runtime:  runtime,
		registry: make(map[SandboxID]*gvisorContainer),
	}
}

// Name returns the adapter name.
func (g *GVisorAdapter) Name() string {
	return "gvisor"
}

// IsolationLevel returns the isolation level for gVisor containers.
func (g *GVisorAdapter) IsolationLevel() string {
	return "container"
}

// Create provisions a new container using Docker with the gVisor (runsc) runtime.
func (g *GVisorAdapter) Create(ctx context.Context, cfg SandboxConfig) (*SandboxResult, error) {
	id := SandboxID(fmt.Sprintf("sbx-gv-%s-%d", cfg.Name, 0))
	args := []string{
		"run", "-d",
		"--runtime=" + g.runtime,
		"--name", string(id),
	}

	if cfg.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", cfg.MemoryMB))
	}
	if cfg.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.1f", cfg.CPUs))
	}
	for _, m := range cfg.Mounts {
		args = append(args, "-v", m)
	}
	for _, e := range cfg.Env {
		args = append(args, "-e", e)
	}
	for _, pm := range cfg.PortMap {
		proto := pm.Protocol
		if proto == "" {
			proto = "tcp"
		}
		args = append(args, "-p", fmt.Sprintf("%d:%d/%s", pm.HostPort, pm.ContainerPort, proto))
	}

	args = append(args, cfg.Image)
	if len(cfg.Command) > 0 {
		args = append(args, cfg.Command...)
	}

	cmd := exec.CommandContext(ctx, g.docker, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gvisor docker create failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	g.mu.Lock()
	g.registry[id] = &gvisorContainer{
		id:     id,
		config: cfg,
		status: SandboxStatusRunning,
	}
	g.mu.Unlock()

	return &SandboxResult{ID: id, Status: SandboxStatusRunning}, nil
}

// Stop sends a stop signal to the gVisor container.
func (g *GVisorAdapter) Stop(ctx context.Context, id SandboxID, force bool) error {
	args := []string{"stop"}
	if force {
		args = append(args, "-t", "0")
	}
	args = append(args, string(id))

	cmd := exec.CommandContext(ctx, g.docker, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gvisor stop failed for %s: %s: %w", id, strings.TrimSpace(string(out)), err)
	}

	g.mu.Lock()
	if sc, ok := g.registry[id]; ok {
		sc.status = SandboxStatusStopped
	}
	g.mu.Unlock()

	return nil
}

// Remove deletes the gVisor container.
func (g *GVisorAdapter) Remove(ctx context.Context, id SandboxID, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, string(id))

	cmd := exec.CommandContext(ctx, g.docker, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gvisor rm failed for %s: %s: %w", id, strings.TrimSpace(string(out)), err)
	}

	g.mu.Lock()
	delete(g.registry, id)
	g.mu.Unlock()

	return nil
}

// Status queries the running status of the gVisor container.
func (g *GVisorAdapter) Status(ctx context.Context, id SandboxID) (*SandboxResult, error) {
	cmd := exec.CommandContext(ctx, g.docker, "inspect", "--format", "{{.State.Status}}", string(id))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gvisor inspect failed for %s: %w", id, err)
	}

	raw := strings.TrimSpace(string(out))
	var status SandboxStatus
	switch raw {
	case "running":
		status = SandboxStatusRunning
	case "exited", "dead":
		status = SandboxStatusStopped
	case "created":
		status = SandboxStatusCreated
	default:
		status = SandboxStatusError
	}

	g.mu.Lock()
	if sc, ok := g.registry[id]; ok {
		sc.status = status
	}
	g.mu.Unlock()

	return &SandboxResult{ID: id, Status: status}, nil
}
