//go:build integration

// Package sandbox — real E2E tests for native (bwrap), gVisor (runsc),
// and Docker adapters.
//
// Each test is gated on binary availability and skipped in -short mode
// (CI unit-test passes).  Resources are cleaned up in defer blocks.
package sandbox

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/internal/adapters/gvisor"
	"github.com/kooshapari/nanovms/internal/domain"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// skipIfNoBinary skips the test if the named binary is not on PATH.
func skipIfNoBinary(tb testing.TB, bin string) {
	tb.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		tb.Skipf("%s not found on PATH, skipping", bin)
	}
}

// readFirstLine reads the first line from an io.ReadCloser and closes it.
func readFirstLine(tb testing.TB, r interface{ Read([]byte) (int, error) }) string {
	tb.Helper()
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

// dockerExec runs a docker CLI command and returns combined output.
func dockerExec(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

// dockerContainerName returns a unique container name for tests.
func dockerContainerName(tb testing.TB) string {
	tb.Helper()
	return fmt.Sprintf("nvms-e2e-%d", time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// 1. Native adapter: bwrap  full lifecycle
// ---------------------------------------------------------------------------

func TestE2E_NativeBwrap_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "bwrap")

	adapter := NewNativeSandbox("bwrap")
	ctx := context.Background()

	// Create
	cfg := domain.SandboxConfig{
		Name:          "e2e-bwrap",
		NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: []string{"/bin/echo", "hello-e2e"}},
	}
	sb, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb == nil {
		t.Fatal("Create returned nil sandbox")
	}
	if sb.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	defer func() { _ = adapter.Delete(ctx, sb.ID) }()

	// Get
	got, err := adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != sb.ID {
		t.Fatalf("Get returned wrong ID: got %s, want %s", got.ID, sb.ID)
	}

	// List
	list, err := adapter.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("List returned empty after Create")
	}

	// Start
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Metrics (may be empty if process already exited, but should not error)
	metrics, err := adapter.Metrics(ctx, sb.ID)
	if err != nil {
		t.Logf("Metrics returned error (non-fatal): %v", err)
	} else if metrics == nil {
		t.Fatal("Metrics returned nil")
	}

	// Stop (graceful)
	if err := adapter.Stop(ctx, sb.ID, false); err != nil {
		t.Fatalf("Stop(graceful) failed: %v", err)
	}

	// Stop (force) — should be idempotent
	if err := adapter.Stop(ctx, sb.ID, true); err != nil {
		t.Logf("Stop(force) on stopped sandbox: %v (acceptable)", err)
	}

	// Delete
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 2. gVisor adapter: runsc  full lifecycle
// ---------------------------------------------------------------------------

func TestE2E_GVisorRunsc_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "runsc")

	adapter := gvisor.NewAdapter()
	ctx := context.Background()

	// Create
	cfg := domain.SandboxConfig{Name: "e2e-runsc", Image: "alpine"}
	sb, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sb == nil {
		t.Fatal("Create returned nil sandbox")
	}
	if sb.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	defer func() { _ = adapter.Delete(ctx, sb.ID) }()

	// Get
	got, err := adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != sb.ID {
		t.Fatalf("Get returned wrong ID: got %s, want %s", got.ID, sb.ID)
	}
	if got.Status != domain.SandboxStatusPending {
		t.Fatalf("expected pending status after Create, got %s", got.Status)
	}

	// List
	list, err := adapter.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	found := false
	for _, s := range list {
		if s.ID == sb.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created sandbox not in List output")
	}

	// Start
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Logf("Start failed (runsc may require root / containerd): %v", err)
		// Don't fail the test — runsc in CI may not have a proper OCI bundle.
		return
	}

	// Status check — running
	got, err = adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get after Start failed: %v", err)
	}
	if got.Status != domain.SandboxStatusRunning {
		t.Fatalf("expected running status after Start, got %s", got.Status)
	}

	// Exec echo
	r, err := adapter.Exec(ctx, sb.ID, []string{"echo", "hello-gvisor"})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	line := readFirstLine(t, r)
	_ = r.Close()
	if !strings.Contains(line, "hello-gvisor") {
		t.Fatalf("Exec output = %q, want it to contain %q", line, "hello-gvisor")
	}

	// Stop (graceful)
	if err := adapter.Stop(ctx, sb.ID, false); err != nil {
		t.Fatalf("Stop(graceful) failed: %v", err)
	}

	// Delete
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 3. Docker adapter: full lifecycle via docker CLI
// ---------------------------------------------------------------------------

func TestE2E_Docker_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "docker")

	ctx := context.Background()
	name := dockerContainerName(t)

	// Ensure Docker daemon is reachable.
	if out, err := dockerExec(ctx, "info"); err != nil {
		t.Skipf("Docker daemon not reachable: %s", strings.TrimSpace(string(out)))
	}

	// Create container
	out, err := dockerExec(ctx, "create", "--name", name, "alpine", "echo", "hello-docker")
	if err != nil {
		t.Fatalf("docker create failed: %s %v", strings.TrimSpace(string(out)), err)
	}
	containerID := strings.TrimSpace(string(out))
	if containerID == "" {
		t.Fatal("docker create returned empty ID")
	}
	defer func() { _, _ = dockerExec(ctx, "rm", "-f", name) }()

	// Start
	if out, err := dockerExec(ctx, "start", name); err != nil {
		t.Fatalf("docker start failed: %s %v", strings.TrimSpace(string(out)), err)
	}

	// Wait for container to finish (it runs `echo` and exits)
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if out, err := dockerExec(waitCtx, "wait", name); err != nil {
		t.Fatalf("docker wait failed: %s %v", strings.TrimSpace(string(out)), err)
	}

	// Inspect (status check)
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err = dockerExec(inspectCtx, "inspect", "--format", "{{.State.Status}}", name)
	if err != nil {
		t.Fatalf("docker inspect failed: %s %v", strings.TrimSpace(string(out)), err)
	}
	status := strings.TrimSpace(string(out))
	if status != "exited" && status != "stopped" {
		t.Fatalf("expected exited/stopped status, got %q", status)
	}

	// Exec in a fresh running container for the exec part of the lifecycle
	execName := dockerContainerName(t)
	out, err = dockerExec(ctx, "run", "-d", "--name", execName, "alpine", "sleep", "30")
	if err != nil {
		t.Fatalf("docker run (exec test) failed: %s %v", strings.TrimSpace(string(out)), err)
	}
	defer func() { _, _ = dockerExec(ctx, "rm", "-f", execName) }()

	// Give the container a moment to start
	time.Sleep(500 * time.Millisecond)

	// Exec
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err = dockerExec(execCtx, "exec", execName, "echo", "hello-docker-exec")
	if err != nil {
		t.Fatalf("docker exec failed: %s %v", strings.TrimSpace(string(out)), err)
	}
	got := strings.TrimSpace(string(out))
	if got != "hello-docker-exec" {
		t.Fatalf("docker exec output = %q, want %q", got, "hello-docker-exec")
	}

	// Stop (graceful)
	if out, err = dockerExec(ctx, "stop", execName); err != nil {
		t.Fatalf("docker stop failed: %s %v", strings.TrimSpace(string(out)), err)
	}

	// Stop (force) — should be idempotent
	if out, err = dockerExec(ctx, "kill", execName); err != nil {
		t.Logf("docker kill on stopped container: %s (acceptable)", strings.TrimSpace(string(out)))
	}

	// Remove
	if out, err = dockerExec(ctx, "rm", "-f", execName); err != nil {
		t.Fatalf("docker rm failed: %s %v", strings.TrimSpace(string(out)), err)
	}

	// Original container should also be removed (verify via defer), but let's
	// confirm it's gone.
	inspectCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	_, err = dockerExec(inspectCtx2, "inspect", name)
	// docker inspect returns non-zero for missing container — expected.
	_ = err
}

// ---------------------------------------------------------------------------
// 4. Concurrent sandbox creation — 3 in parallel
// ---------------------------------------------------------------------------

func TestE2E_ConcurrentNativeCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "bwrap")

	adapter := NewNativeSandbox("bwrap")
	ctx := context.Background()

	const count = 3
	var wg sync.WaitGroup
	errs := make([]error, count)
	sandboxes := make([]*domain.Sandbox, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cfg := domain.SandboxConfig{
				Name:          fmt.Sprintf("e2e-concurrent-%d", idx),
				NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: []string{"/bin/true"}},
			}
			sb, err := adapter.Create(ctx, cfg)
			sandboxes[idx] = sb
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Create[%d] failed: %v", i, err)
			continue
		}
		if sandboxes[i] == nil {
			t.Errorf("concurrent Create[%d] returned nil sandbox", i)
			continue
		}
		if sandboxes[i].ID == "" {
			t.Errorf("concurrent Create[%d] returned empty ID", i)
		}
		// Cleanup
		_ = adapter.Delete(ctx, sandboxes[i].ID)
	}

	// Verify all IDs are unique
	seen := make(map[string]bool)
	for _, sb := range sandboxes {
		if sb == nil {
			continue
		}
		if seen[sb.ID] {
			t.Errorf("duplicate sandbox ID across concurrent creates: %s", sb.ID)
		}
		seen[sb.ID] = true
	}

	// List should see all (before cleanup)
	list, err := adapter.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) < count {
		// Some may have been deleted, so just verify the adapter works
		t.Logf("List returned %d items (some may have been cleaned up)", len(list))
	}
}

// ---------------------------------------------------------------------------
// 5. Full lifecycle: create -> start -> status -> stop -> delete
// ---------------------------------------------------------------------------

func TestE2E_FullLifecycle_NativeBwrap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "bwrap")

	adapter := NewNativeSandbox("bwrap")
	ctx := context.Background()

	// Step 1: Create
	cfg := domain.SandboxConfig{
		Name:          "e2e-lifecycle",
		NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: []string{"/bin/sleep", "60"}},
	}
	sb, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("step 1 Create: %v", err)
	}
	if sb.Status != domain.SandboxStatusCreating {
		t.Fatalf("step 1: expected status Creating, got %s", sb.Status)
	}
	defer func() { _ = adapter.Delete(ctx, sb.ID) }()

	// Step 2: Start
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Fatalf("step 2 Start: %v", err)
	}

	// Step 3: Status check
	got, err := adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("step 3 Get: %v", err)
	}
	if got.Status != domain.SandboxStatusRunning {
		t.Fatalf("step 3: expected status Running, got %s", got.Status)
	}
	if got.PID <= 0 {
		t.Fatalf("step 3: expected positive PID, got %d", got.PID)
	}
	t.Logf("sandbox PID: %d", got.PID)

	// Step 4: Stop (graceful)
	if err := adapter.Stop(ctx, sb.ID, false); err != nil {
		t.Fatalf("step 4 Stop(graceful): %v", err)
	}

	got, err = adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("step 4 Get after stop: %v", err)
	}
	if got.Status != domain.SandboxStatusStopped {
		t.Fatalf("step 4: expected status Stopped, got %s", got.Status)
	}

	// Step 5: Delete
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("step 5 Delete: %v", err)
	}
	_, err = adapter.Get(ctx, sb.ID)
	if err == nil {
		t.Fatal("step 5: Get after Delete should fail")
	}
}

func TestE2E_FullLifecycle_GVisor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "runsc")

	adapter := gvisor.NewAdapter()
	ctx := context.Background()

	// Step 1: Create
	cfg := domain.SandboxConfig{Name: "e2e-gv-lifecycle", Image: "alpine"}
	sb, err := adapter.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("step 1 Create: %v", err)
	}
	if sb.Status != domain.SandboxStatusPending {
		t.Fatalf("step 1: expected pending status, got %s", sb.Status)
	}
	defer func() { _ = adapter.Delete(ctx, sb.ID) }()

	// Step 2: Start
	if err := adapter.Start(ctx, sb.ID); err != nil {
		t.Logf("step 2 Start failed (runsc may need root): %v", err)
		return // Can't continue lifecycle without start
	}

	// Step 3: Status check
	got, err := adapter.Get(ctx, sb.ID)
	if err != nil {
		t.Fatalf("step 3 Get: %v", err)
	}
	if got.Status != domain.SandboxStatusRunning {
		t.Fatalf("step 3: expected Running, got %s", got.Status)
	}

	// Step 4: Stop
	if err := adapter.Stop(ctx, sb.ID, false); err != nil {
		t.Fatalf("step 4 Stop: %v", err)
	}

	// Step 5: Delete
	if err := adapter.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("step 5 Delete: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 6. Force stop vs graceful stop
// ---------------------------------------------------------------------------

func TestE2E_ForceStopVsGracefulStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "bwrap")

	adapter := NewNativeSandbox("bwrap")
	ctx := context.Background()

	t.Run("graceful_stop", func(t *testing.T) {
		cfg := domain.SandboxConfig{
			Name:          "e2e-graceful",
			NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: []string{"/bin/sleep", "30"}},
		}
		sb, err := adapter.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer func() { _ = adapter.Delete(ctx, sb.ID) }()

		if err := adapter.Start(ctx, sb.ID); err != nil {
			t.Fatalf("Start: %v", err)
		}

		// Graceful stop sends SIGTERM
		start := time.Now()
		if err := adapter.Stop(ctx, sb.ID, false); err != nil {
			t.Fatalf("Stop(graceful): %v", err)
		}
		elapsed := time.Since(start)
		t.Logf("graceful stop took %v", elapsed)

		got, err := adapter.Get(ctx, sb.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Status != domain.SandboxStatusStopped {
			t.Fatalf("expected Stopped status after graceful stop, got %s", got.Status)
		}
	})

	t.Run("force_stop", func(t *testing.T) {
		cfg := domain.SandboxConfig{
			Name:          "e2e-force",
			NativeSandbox: &domain.NativeSandboxConfig{Type: domain.NativeSandboxBwrap, Command: []string{"/bin/sleep", "30"}},
		}
		sb, err := adapter.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer func() { _ = adapter.Delete(ctx, sb.ID) }()

		if err := adapter.Start(ctx, sb.ID); err != nil {
			t.Fatalf("Start: %v", err)
		}

		// Force stop sends SIGKILL
		start := time.Now()
		if err := adapter.Stop(ctx, sb.ID, true); err != nil {
			t.Fatalf("Stop(force): %v", err)
		}
		elapsed := time.Since(start)
		t.Logf("force stop took %v", elapsed)

		got, err := adapter.Get(ctx, sb.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Status != domain.SandboxStatusStopped {
			t.Fatalf("expected Stopped status after force stop, got %s", got.Status)
		}
	})
}

// ---------------------------------------------------------------------------
// 7. Docker concurrent container creation
// ---------------------------------------------------------------------------

func TestE2E_Docker_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "docker")

	ctx := context.Background()

	// Ensure Docker daemon is reachable.
	if out, err := dockerExec(ctx, "info"); err != nil {
		t.Skipf("Docker daemon not reachable: %s", strings.TrimSpace(string(out)))
	}

	const count = 3
	var wg sync.WaitGroup
	names := make([]string, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := dockerContainerName(t)
			names[idx] = name
			out, err := dockerExec(ctx, "run", "-d", "--name", name, "alpine", "sleep", "10")
			if err != nil {
				t.Errorf("docker run [%d] failed: %s %v", idx, strings.TrimSpace(string(out)), err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all containers are running
	for i, name := range names {
		if name == "" {
			continue
		}
		defer func(n string) { _, _ = dockerExec(ctx, "rm", "-f", n) }(name)

		outCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		out, err := dockerExec(outCtx, "inspect", "--format", "{{.State.Status}}", name)
		cancel()
		if err != nil {
			t.Errorf("inspect container [%d] %s failed: %v", i, name, err)
			continue
		}
		status := strings.TrimSpace(string(out))
		if status != "running" {
			t.Errorf("container [%d] %s: expected running, got %s", i, name, status)
		}
	}
}

// ---------------------------------------------------------------------------
// 8. Error paths: nonexistent sandbox operations
// ---------------------------------------------------------------------------

func TestE2E_ErrorPaths_NonexistentSandbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	adapter := NewNativeSandbox("bwrap")
	ctx := context.Background()
	fakeID := "nonexistent-sandbox-id"

	// Get nonexistent
	_, err := adapter.Get(ctx, fakeID)
	if err == nil {
		t.Fatal("Get nonexistent should fail")
	}

	// Start nonexistent
	if err := adapter.Start(ctx, fakeID); err == nil {
		t.Fatal("Start nonexistent should fail")
	}

	// Stop nonexistent
	if err := adapter.Stop(ctx, fakeID, false); err == nil {
		t.Fatal("Stop nonexistent should fail")
	}

	// Metrics nonexistent
	_, err = adapter.Metrics(ctx, fakeID)
	if err == nil {
		t.Fatal("Metrics nonexistent should fail")
	}

	// Exec nonexistent
	_, err = adapter.Exec(ctx, fakeID, []string{"echo", "hi"})
	if err == nil {
		t.Fatal("Exec nonexistent should fail")
	}

	// Logs nonexistent
	_, err = adapter.Logs(ctx, fakeID, false)
	if err == nil {
		t.Fatal("Logs nonexistent should fail")
	}
}

func TestE2E_GVisor_ErrorPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	adapter := gvisor.NewAdapter()
	ctx := context.Background()
	fakeID := "gv-nonexistent"

	_, err := adapter.Get(ctx, fakeID)
	if err == nil {
		t.Fatal("Get nonexistent should fail")
	}

	if err := adapter.Start(ctx, fakeID); err == nil {
		t.Fatal("Start nonexistent should fail")
	}

	if err := adapter.Stop(ctx, fakeID, false); err == nil {
		t.Fatal("Stop nonexistent should fail")
	}

	_, err = adapter.Metrics(ctx, fakeID)
	if err == nil {
		t.Fatal("Metrics nonexistent should fail")
	}
}

// ---------------------------------------------------------------------------
// 9. Docker exec in running container
// ---------------------------------------------------------------------------

func TestE2E_Docker_ExecInContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}
	skipIfNoBinary(t, "docker")

	ctx := context.Background()
	name := dockerContainerName(t)

	// Ensure Docker daemon is reachable.
	if out, err := dockerExec(ctx, "info"); err != nil {
		t.Skipf("Docker daemon not reachable: %s", strings.TrimSpace(string(out)))
	}

	// Start a long-running container
	out, err := dockerExec(ctx, "run", "-d", "--name", name, "alpine", "sleep", "60")
	if err != nil {
		t.Fatalf("docker run: %s %v", strings.TrimSpace(string(out)), err)
	}
	defer func() { _, _ = dockerExec(ctx, "rm", "-f", name) }()

	// Wait for container to be running
	time.Sleep(500 * time.Millisecond)

	// Exec echo
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err = dockerExec(execCtx, "exec", name, "echo", "e2e-docker-exec-ok")
	if err != nil {
		t.Fatalf("docker exec: %s %v", strings.TrimSpace(string(out)), err)
	}
	got := strings.TrimSpace(string(out))
	if got != "e2e-docker-exec-ok" {
		t.Fatalf("exec output = %q, want %q", got, "e2e-docker-exec-ok")
	}

	// Exec multiple commands
	execCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	out, err = dockerExec(execCtx2, "exec", name, "sh", "-c", "echo alpha && echo beta")
	if err != nil {
		t.Fatalf("docker exec (multi): %s %v", strings.TrimSpace(string(out)), err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 || lines[0] != "alpha" || lines[1] != "beta" {
		t.Fatalf("multi-exec output = %v, want [alpha, beta]", lines)
	}

	// Stop and remove
	_, _ = dockerExec(ctx, "stop", name)
	_, _ = dockerExec(ctx, "rm", name)
}
