//go:build integration
// +build integration

// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- real Docker integration tests.
//
// These tests exercise the DockerAdapter against a live Docker daemon.
// Every test is skipped when:
//   - the "integration" build tag is absent (normal `go test` without -tags)
//   - testing.Short() is true (`go test -short`)
//   - the `docker` CLI is not on PATH
//   - the Docker daemon is not reachable
//
// Run with:
//
//	go test -tags=integration -count=1 -v ./internal/adapters/sandbox/ -run TestDockerReal
//	go test -tags=integration -short ./internal/adapters/sandbox/   # skipped

package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// dockerRealSkip skips the current test unless Docker is fully available.
func dockerRealSkip(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping real Docker integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI not found on PATH; skipping")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput(); err != nil {
		t.Skipf("docker daemon not reachable: %s (%v)", strings.TrimSpace(string(out)), err)
	}
}

// realDockerCleanup is a best-effort teardown for a container ID.
func realDockerCleanup(t *testing.T, id SandboxID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "docker", "stop", "-t", "0", string(id)).Run()
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", string(id)).Run()
}

// TestDockerRealCreateAndInspect creates a real container and verifies it
// appears in `docker inspect` with the expected configuration.
func TestDockerRealCreateAndInspect(t *testing.T) {
	dockerRealSkip(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const image = "busybox:latest"

	cfg := SandboxConfig{
		Name:     "int-real-create",
		Image:    image,
		Command:  []string{"sleep", "300"},
		MemoryMB: 32,
		CPUs:     0.25,
		Env:      []string{"NANOVMS_TEST=1"},
	}

	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer realDockerCleanup(t, result.ID)

	t.Logf("created container %s (status=%s)", result.ID, result.Status)

	// Verify docker inspect can find it.
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Name}}", string(result.ID)).Output()
	if err != nil {
		t.Fatalf("docker inspect: %v", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		t.Fatal("docker inspect returned empty name")
	}
	t.Logf("inspected container name: %s", name)

	// Verify memory limit via inspect.
	out, err = exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{.HostConfig.Memory}}", string(result.ID)).Output()
	if err != nil {
		t.Fatalf("docker inspect memory: %v", err)
	}
	mem := strings.TrimSpace(string(out))
	if mem != "33554432" { // 32 MB = 33554432 bytes
		t.Fatalf("expected memory 33554432, got %s", mem)
	}
	t.Logf("memory constraint verified: %s bytes", mem)
}

// TestDockerRealStartStopStart exercises the docker start -> stop -> start
// cycle through the adapter, verifying status transitions at each step.
func TestDockerRealStartStopStart(t *testing.T) {
	dockerRealSkip(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:    "int-real-start-stop",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
	}

	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer realDockerCleanup(t, result.ID)

	// Start
	if err := exec.CommandContext(ctx, "docker", "start", string(result.ID)).Run(); err != nil {
		t.Fatalf("docker start: %v", err)
	}

	s, err := adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after start: %v", err)
	}
	if s.Status != SandboxStatusRunning {
		t.Fatalf("expected running, got %s", s.Status)
	}

	// Stop
	if err := adapter.Stop(ctx, result.ID, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	s, err = adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after stop: %v", err)
	}
	if s.Status != SandboxStatusStopped {
		t.Fatalf("expected stopped, got %s", s.Status)
	}

	// Restart
	if err := exec.CommandContext(ctx, "docker", "start", string(result.ID)).Run(); err != nil {
		t.Fatalf("docker restart: %v", err)
	}

	s, err = adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after restart: %v", err)
	}
	if s.Status != SandboxStatusRunning {
		t.Fatalf("expected running after restart, got %s", s.Status)
	}

	t.Log("start -> stop -> start lifecycle verified")
}

// TestDockerRealForceKillAndRemove verifies that a running container can be
// force-stopped and removed in a single Remove(force=true) call.
func TestDockerRealForceKillAndRemove(t *testing.T) {
	dockerRealSkip(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:    "int-real-force",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
	}
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer realDockerCleanup(t, result.ID)

	// Start the container.
	if err := exec.CommandContext(ctx, "docker", "start", string(result.ID)).Run(); err != nil {
		t.Fatalf("docker start: %v", err)
	}

	// Verify it is running.
	s, err := adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if s.Status != SandboxStatusRunning {
		t.Fatalf("expected running, got %s", s.Status)
	}

	// Force remove.
	if err := adapter.Remove(ctx, result.ID, true); err != nil {
		t.Fatalf("Remove(force): %v", err)
	}

	// Confirm gone.
	_, err = adapter.Status(ctx, result.ID)
	if err == nil {
		t.Fatal("Status should fail after force Remove")
	}
	t.Log("force kill+remove verified")
}

// TestDockerRealMultipleContainers verifies that multiple containers can be
// created, tracked, and cleaned up independently.
func TestDockerRealMultipleContainers(t *testing.T) {
	dockerRealSkip(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const n = 3
	var ids []SandboxID
	for i := 0; i < n; i++ {
		cfg := SandboxConfig{
			Name:    "int-real-multi",
			Image:   "busybox:latest",
			Command: []string{"sleep", "300"},
		}
		result, err := adapter.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		ids = append(ids, result.ID)
	}

	// Ensure cleanup.
	defer func() {
		for _, id := range ids {
			realDockerCleanup(t, id)
		}
	}()

	// Start all.
	for _, id := range ids {
		if err := exec.CommandContext(ctx, "docker", "start", string(id)).Run(); err != nil {
			t.Fatalf("docker start %s: %v", id, err)
		}
	}

	// Verify all running.
	for _, id := range ids {
		s, err := adapter.Status(ctx, id)
		if err != nil {
			t.Fatalf("Status %s: %v", id, err)
		}
		if s.Status != SandboxStatusRunning {
			t.Fatalf("container %s: expected running, got %s", id, s.Status)
		}
	}

	// Stop all.
	for _, id := range ids {
		if err := adapter.Stop(ctx, id, false); err != nil {
			t.Fatalf("Stop %s: %v", id, err)
		}
	}

	// Verify all stopped.
	for _, id := range ids {
		s, err := adapter.Status(ctx, id)
		if err != nil {
			t.Fatalf("Status %s: %v", id, err)
		}
		if s.Status != SandboxStatusStopped {
			t.Fatalf("container %s: expected stopped, got %s", id, s.Status)
		}
	}

	// Remove all.
	for _, id := range ids {
		if err := adapter.Remove(ctx, id, false); err != nil {
			t.Fatalf("Remove %s: %v", id, err)
		}
	}

	t.Logf("created, started, stopped, and removed %d containers", n)
}

// TestDockerRealPortMapping verifies that port mappings are applied.
func TestDockerRealPortMapping(t *testing.T) {
	dockerRealSkip(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const hostPort = 19877

	cfg := SandboxConfig{
		Name:    "int-real-port",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
		PortMap: []PortMapping{
			{HostPort: hostPort, ContainerPort: 80, Protocol: "tcp"},
		},
	}
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer realDockerCleanup(t, result.ID)

	// Start the container.
	if err := exec.CommandContext(ctx, "docker", "start", string(result.ID)).Run(); err != nil {
		t.Fatalf("docker start: %v", err)
	}

	// Inspect the port bindings.
	out, err := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{{index .HostConfig.PortBindings "80/tcp" 0 "HostPort"}}`,
		string(result.ID)).Output()
	if err != nil {
		t.Fatalf("docker inspect port: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "19877" {
		t.Fatalf("expected host port 19877, got %q", got)
	}
	t.Logf("port mapping verified: host %s -> container 80/tcp", got)
}
