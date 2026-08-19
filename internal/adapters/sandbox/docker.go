// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- Docker adapter using os/exec for container management.

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// SandboxConfig holds configuration for creating a Docker container.
type SandboxConfig struct {
	Name        string
	Image       string
	Command     []string
	MemoryMB    int
	CPUs        float64
	Mounts      []string
	PortMap     []PortMapping
	Env         []string
	NetworkMode string
	Privileged  bool
}

// PortMapping defines a host-to-container port mapping.
type PortMapping struct {
	HostPort      int
	ContainerPort int
	Protocol      string // "tcp" or "udp"
}

// SandboxID is a typed identifier for a Docker container.
type SandboxID string

// SandboxStatus represents the lifecycle state of a container.
type SandboxStatus string

const (
	SandboxStatusCreated SandboxStatus = "created"
	SandboxStatusRunning SandboxStatus = "running"
	SandboxStatusStopped SandboxStatus = "stopped"
	SandboxStatusError   SandboxStatus = "error"
)

// SandboxResult holds the result of a Docker container operation.
type SandboxResult struct {
	ID     SandboxID
	Status SandboxStatus
	Error  error
}

// DockerAdapter implements real Docker container operations via os/exec.
type DockerAdapter struct {
	mu       sync.Mutex
	client   string
	registry map[SandboxID]*dockerContainer
}

type dockerContainer struct {
	id     SandboxID
	config SandboxConfig
	status SandboxStatus
}

// NewDockerAdapter creates a new Docker adapter that shells out to the docker CLI.
func NewDockerAdapter() *DockerAdapter {
	client := "docker"
	if p, err := exec.LookPath("docker"); err == nil {
		client = p
	}
	return &DockerAdapter{
		client:   client,
		registry: make(map[SandboxID]*dockerContainer),
	}
}

// Name returns the adapter name.
func (d *DockerAdapter) Name() string {
	return "docker"
}

// IsolationLevel returns the isolation level for Docker containers.
func (d *DockerAdapter) IsolationLevel() string {
	return "container"
}

// Create provisions a new Docker container from the provided configuration.
func (d *DockerAdapter) Create(ctx context.Context, cfg SandboxConfig) (*SandboxResult, error) {
	id := SandboxID(fmt.Sprintf("sbx-%s-%d", cfg.Name, 0))
	args := []string{"create", "--name", string(id)}

	if cfg.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", cfg.MemoryMB))
	}
	if cfg.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.1f", cfg.CPUs))
	}
	if cfg.Privileged {
		args = append(args, "--privileged")
	}
	if cfg.NetworkMode != "" {
		args = append(args, "--network", cfg.NetworkMode)
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

	cmd := exec.CommandContext(ctx, d.client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker create failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	d.mu.Lock()
	d.registry[id] = &dockerContainer{
		id:     id,
		config: cfg,
		status: SandboxStatusCreated,
	}
	d.mu.Unlock()

	return &SandboxResult{ID: id, Status: SandboxStatusCreated}, nil
}

// Stop gracefully stops a running container, optionally killing it.
func (d *DockerAdapter) Stop(ctx context.Context, id SandboxID, force bool) error {
	args := []string{"stop"}
	if force {
		args = append(args, "-t", "0")
	}
	args = append(args, string(id))

	cmd := exec.CommandContext(ctx, d.client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed for %s: %s: %w", id, strings.TrimSpace(string(out)), err)
	}

	d.mu.Lock()
	if sc, ok := d.registry[id]; ok {
		sc.status = SandboxStatusStopped
	}
	d.mu.Unlock()

	return nil
}

// Remove deletes a container (must be stopped first unless force is true).
func (d *DockerAdapter) Remove(ctx context.Context, id SandboxID, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, string(id))

	cmd := exec.CommandContext(ctx, d.client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed for %s: %s: %w", id, strings.TrimSpace(string(out)), err)
	}

	d.mu.Lock()
	delete(d.registry, id)
	d.mu.Unlock()

	return nil
}

// Status queries the running status of a container via docker inspect.
func (d *DockerAdapter) Status(ctx context.Context, id SandboxID) (*SandboxResult, error) {
	cmd := exec.CommandContext(ctx, d.client, "inspect", "--format", "{{.State.Status}}", string(id))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed for %s: %w", id, err)
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

	d.mu.Lock()
	if sc, ok := d.registry[id]; ok {
		sc.status = status
	}
	d.mu.Unlock()

	return &SandboxResult{ID: id, Status: status}, nil
}
