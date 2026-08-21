//go:build integration
// +build integration

// SPDX-License-Identifier: MIT OR Apache-2.0
// Package sandbox -- integration tests for Docker, gVisor, and Firecracker adapters.
//
// These tests exercise real runtime operations and require the corresponding
// binaries (docker, runsc, firecracker) on PATH. They are gated behind the
// "integration" build tag and testing.Short() so they never execute during
// normal `go test` runs.
//
// Run with:
//
//	go test -tags=integration -count=1 -v ./internal/adapters/sandbox/ -run TestDocker
//	go test -tags=integration -count=1 -v ./internal/adapters/sandbox/ -run TestGVisor
//	go test -tags=integration -count=1 -v ./internal/adapters/sandbox/ -run TestFirecracker
//	go test -tags=integration -short ./internal/adapters/sandbox/   # all skipped

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// requireDocker skips the test if docker is not available or the daemon is
// unreachable.
func requireDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
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

// requireRunsc skips the test if runsc (gVisor) is not on PATH.
func requireRunsc(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("runsc"); err != nil {
		t.Skip("runsc (gVisor) not found on PATH; skipping")
	}
}

// requireFirecracker skips the test if firecracker is not on PATH.
func requireFirecracker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("firecracker"); err != nil {
		t.Skip("firecracker not found on PATH; skipping")
	}
}

// dockerCleanup forcefully stops and removes a container, logging but never
// failing (intended for deferred teardown).
func dockerCleanup(t *testing.T, client string, id SandboxID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, client, "stop", "-t", "0", string(id)).Run()
	_ = exec.CommandContext(ctx, client, "rm", "-f", string(id)).Run()
}

// ===========================================================================
// DockerAdapter -- full lifecycle integration
// ===========================================================================

func TestDockerAdapterCreateStopRemoveLifecycle(t *testing.T) {
	requireDocker(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const image = "busybox:latest"

	cfg := SandboxConfig{
		Name:    "int-docker-lifecycle",
		Image:   image,
		Command: []string{"sleep", "300"},
	}

	// -- Create --
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Status != SandboxStatusCreated {
		t.Fatalf("expected status %q, got %q", SandboxStatusCreated, result.Status)
	}
	t.Logf("created container %s", result.ID)
	defer dockerCleanup(t, adapter.client, result.ID)

	// -- Status (created) --
	s, err := adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after create: %v", err)
	}
	if s.Status != SandboxStatusCreated {
		t.Fatalf("expected %q, got %q", SandboxStatusCreated, s.Status)
	}

	// -- Start via docker CLI --
	if out, err := exec.CommandContext(ctx, adapter.client, "start", string(result.ID)).CombinedOutput(); err != nil {
		t.Fatalf("docker start: %s: %v", strings.TrimSpace(string(out)), err)
	}

	// -- Status (running) --
	s, err = adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after start: %v", err)
	}
	if s.Status != SandboxStatusRunning {
		t.Fatalf("expected %q, got %q", SandboxStatusRunning, s.Status)
	}

	// -- Stop --
	if err := adapter.Stop(ctx, result.ID, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// -- Status (stopped) --
	s, err = adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after stop: %v", err)
	}
	if s.Status != SandboxStatusStopped {
		t.Fatalf("expected %q, got %q", SandboxStatusStopped, s.Status)
	}

	// -- Remove --
	if err := adapter.Remove(ctx, result.ID, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// -- Status (should fail) --
	_, err = adapter.Status(ctx, result.ID)
	if err == nil {
		t.Fatal("Status should fail after Remove")
	}
}

func TestDockerAdapterForceRemoveRunning(t *testing.T) {
	requireDocker(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:    "int-docker-force",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
	}
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer dockerCleanup(t, adapter.client, result.ID)

	// Start
	if out, err := exec.CommandContext(ctx, adapter.client, "start", string(result.ID)).CombinedOutput(); err != nil {
		t.Fatalf("docker start: %s: %v", strings.TrimSpace(string(out)), err)
	}

	// Force remove while running
	if err := adapter.Remove(ctx, result.ID, true); err != nil {
		t.Fatalf("Remove(force): %v", err)
	}

	_, err = adapter.Status(ctx, result.ID)
	if err == nil {
		t.Fatal("Status should fail after force Remove")
	}
}

func TestDockerAdapterCreateWithConstraints(t *testing.T) {
	requireDocker(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:     "int-docker-constraints",
		Image:    "busybox:latest",
		Command:  []string{"sleep", "60"},
		MemoryMB: 64,
		CPUs:     0.5,
		Env:      []string{"INTEGRATION_TEST=1"},
		PortMap:  []PortMapping{{HostPort: 19876, ContainerPort: 80, Protocol: "tcp"}},
	}
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer dockerCleanup(t, adapter.client, result.ID)

	// Verify memory constraint (64 MB = 67108864 bytes).
	out, err := exec.CommandContext(ctx, adapter.client,
		"inspect", "--format", "{{.HostConfig.Memory}}", string(result.ID)).Output()
	if err != nil {
		t.Fatalf("inspect memory: %v", err)
	}
	mem := strings.TrimSpace(string(out))
	if mem != "67108864" {
		t.Fatalf("expected memory 67108864, got %s", mem)
	}
	t.Logf("memory constraint verified: %s bytes", mem)
}

func TestDockerAdapterRegistryConsistency(t *testing.T) {
	requireDocker(t)

	adapter := NewDockerAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:    "int-docker-registry",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
	}
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer dockerCleanup(t, adapter.client, result.ID)

	// After Create, registry should have the entry.
	adapter.mu.Lock()
	_, ok := adapter.registry[result.ID]
	adapter.mu.Unlock()
	if !ok {
		t.Fatal("registry missing entry after Create")
	}

	// Start, stop, then remove -- entry should be deleted.
	if out, err := exec.CommandContext(ctx, adapter.client, "start", string(result.ID)).CombinedOutput(); err != nil {
		t.Fatalf("docker start: %s: %v", strings.TrimSpace(string(out)), err)
	}
	if err := adapter.Stop(ctx, result.ID, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if out, err := exec.CommandContext(ctx, adapter.client, "start", string(result.ID)).CombinedOutput(); err != nil {
		t.Fatalf("docker restart: %s: %v", strings.TrimSpace(string(out)), err)
	}
	if err := adapter.Remove(ctx, result.ID, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	adapter.mu.Lock()
	_, ok = adapter.registry[result.ID]
	adapter.mu.Unlock()
	if ok {
		t.Fatal("registry still contains entry after Remove")
	}
}

// ===========================================================================
// GVisorAdapter integration tests
// ===========================================================================

func TestGVisorAdapterRuntimeDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	path, err := exec.LookPath("runsc")
	if err != nil {
		t.Skip("runsc not found; skipping")
	}

	adapter := NewGVisorAdapter()
	if adapter.runtime == "" {
		t.Fatal("runtime path is empty after construction")
	}

	// Fetch version to confirm runsc responds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, adapter.runtime, "--version").Output()
	if err != nil {
		t.Fatalf("runsc --version: %v", err)
	}
	ver := strings.TrimSpace(string(out))
	if ver == "" {
		t.Fatal("runsc --version produced empty output")
	}
	t.Logf("runsc: path=%s version=%s", path, ver)
}

func TestGVisorAdapterCreateWithRunsc(t *testing.T) {
	requireDocker(t)
	requireRunsc(t)

	adapter := NewGVisorAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg := SandboxConfig{
		Name:     "int-gvisor-create",
		Image:    "busybox:latest",
		Command:  []string{"sleep", "300"},
		MemoryMB: 128,
		CPUs:     1.0,
	}

	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Status != SandboxStatusRunning {
		t.Fatalf("expected running, got %s", result.Status)
	}
	t.Logf("created gVisor container %s", result.ID)
	defer func() {
		_ = adapter.Stop(ctx, result.ID, true)
		_ = adapter.Remove(ctx, result.ID, true)
	}()

	// Status check.
	s, err := adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if s.Status != SandboxStatusRunning {
		t.Fatalf("expected running, got %s", s.Status)
	}

	// Verify runtime was applied via docker inspect.
	out, err := exec.CommandContext(ctx, adapter.docker,
		"inspect", "--format", "{{.HostConfig.Runtime}}", string(result.ID)).Output()
	if err != nil {
		t.Fatalf("inspect runtime: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != adapter.runtime {
		t.Fatalf("runtime mismatch: expected %q, got %q", adapter.runtime, got)
	}
	t.Logf("gVisor runtime verified: %s", got)

	// Stop + Remove.
	if err := adapter.Stop(ctx, result.ID, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := adapter.Remove(ctx, result.ID, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestGVisorAdapterForceRemove(t *testing.T) {
	requireDocker(t)
	requireRunsc(t)

	adapter := NewGVisorAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := adapter.Create(ctx, SandboxConfig{
		Name:    "int-gvisor-force",
		Image:   "busybox:latest",
		Command: []string{"sleep", "300"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := adapter.Remove(ctx, result.ID, true); err != nil {
		t.Fatalf("Remove(force): %v", err)
	}
	_, err = adapter.Status(ctx, result.ID)
	if err == nil {
		t.Fatal("Status should fail after force Remove")
	}
}

// ===========================================================================
// FirecrackerAdapter integration tests
// ===========================================================================

func TestFirecrackerAdapterCapabilities(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	adapter := NewFirecrackerAdapter()

	if adapter.Name() != "firecracker" {
		t.Fatalf("Name() = %q, want %q", adapter.Name(), "firecracker")
	}
	if adapter.IsolationLevel() != "vm" {
		t.Fatalf("IsolationLevel() = %q, want %q", adapter.IsolationLevel(), "vm")
	}
	if adapter.binary == "" {
		t.Fatal("binary field is empty")
	}

	if path, err := exec.LookPath("firecracker"); err == nil {
		t.Logf("firecracker found at %s", path)
	} else {
		t.Logf("firecracker not on PATH (adapter binary=%q)", adapter.binary)
	}
}

func TestFirecrackerAdapterCreateStopRemoveLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	adapter := NewFirecrackerAdapter()
	ctx := context.Background()

	cfg := SandboxConfig{Name: "int-fc-lifecycle"}

	// Create
	result, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Status != SandboxStatusCreated {
		t.Fatalf("expected created, got %s", result.Status)
	}
	t.Logf("created Firecracker VM %s", result.ID)

	// Status (created)
	s, err := adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if s.Status != SandboxStatusCreated {
		t.Fatalf("expected created, got %s", s.Status)
	}

	// Stop
	if err := adapter.Stop(ctx, result.ID, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Status (stopped)
	s, err = adapter.Status(ctx, result.ID)
	if err != nil {
		t.Fatalf("Status after stop: %v", err)
	}
	if s.Status != SandboxStatusStopped {
		t.Fatalf("expected stopped, got %s", s.Status)
	}

	// Remove
	if err := adapter.Remove(ctx, result.ID, false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Status (should fail)
	_, err = adapter.Status(ctx, result.ID)
	if err == nil {
		t.Fatal("Status should fail after Remove")
	}
}

func TestFirecrackerAdapterRegistryConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	adapter := NewFirecrackerAdapter()
	ctx := context.Background()

	const count = 20
	var wg sync.WaitGroup
	errs := make(chan error, count)
	ids := make(chan SandboxID, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cfg := SandboxConfig{Name: fmt.Sprintf("int-fc-concurrent-%d", n)}
			res, err := adapter.Create(ctx, cfg)
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: Create: %w", n, err)
				return
			}
			if res.Status != SandboxStatusCreated {
				errs <- fmt.Errorf("goroutine %d: expected created, got %s", n, res.Status)
				return
			}
			ids <- res.ID
		}(i)
	}

	// Wait for all goroutines and close channels.
	go func() {
		wg.Wait()
		close(errs)
		close(ids)
	}()

	var created []SandboxID
	for e := range errs {
		t.Error(e)
	}
	for id := range ids {
		created = append(created, id)
	}

	if t.Failed() {
		t.FailNow()
	}

	// Verify all IDs are unique.
	seen := make(map[SandboxID]bool, len(created))
	for _, id := range created {
		if seen[id] {
			t.Errorf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
	t.Logf("created %d unique Firecracker VMs concurrently", len(created))

	// Stop and remove all.
	for _, id := range created {
		if err := adapter.Stop(ctx, id, true); err != nil {
			t.Errorf("Stop %s: %v", id, err)
		}
		if err := adapter.Remove(ctx, id, true); err != nil {
			t.Errorf("Remove %s: %v", id, err)
		}
	}
}
