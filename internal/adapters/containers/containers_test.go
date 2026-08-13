package containers

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

type fakeRunner struct {
	calls [][]string
	run   func([]string) ([]byte, error)
}

func (f *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.run == nil {
		return nil, nil
	}
	return f.run(args)
}

func (f *fakeRunner) Stream(_ context.Context, _ string, args ...string) (io.ReadCloser, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	return io.NopCloser(strings.NewReader("stream")), nil
}

func TestAppleLifecycleUsesStoppedCreateAndAppleInspectShape(t *testing.T) {
	const inspect = `[{"id":"apple-id","configuration":{"image":{"reference":"alpine:latest"},"labels":{"owner":"test"}},"status":{"state":"stopped"}}]`
	runner := &fakeRunner{run: func(args []string) ([]byte, error) {
		switch {
		case strings.Join(args, " ") == "system version --format json":
			return []byte(`{"version":"1.0"}`), nil
		case len(args) > 0 && args[0] == "create":
			return []byte("apple-id\n"), nil
		case len(args) > 0 && args[0] == "inspect":
			return []byte(inspect), nil
		default:
			return nil, nil
		}
	}}
	adapter := NewAdapterWithRunner(KindApple, "container", runner)
	cfg := domain.SandboxConfig{
		Name:        "demo",
		Image:       "alpine:latest",
		Environment: map[string]string{"B": "2", "A": "1"},
		Labels:      map[string]string{"owner": "test"},
	}
	sb, err := adapter.Create(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sb.ID != "apple-id" || sb.Status != domain.SandboxStatusStopped || sb.Config.Image != "alpine:latest" {
		t.Fatalf("unexpected sandbox: %#v", sb)
	}
	if err := adapter.Start(context.Background(), sb.ID); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Delete(context.Background(), sb.ID); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(runner.calls[1], " "), "create --name demo --env A=1 --env B=2 --label owner=test alpine:latest"; got != want {
		t.Fatalf("create argv=%q, want %q", got, want)
	}
	if got := strings.Join(runner.calls[len(runner.calls)-1], " "); got != "delete --force apple-id" {
		t.Fatalf("delete argv=%q", got)
	}
}

func TestWSLDialectUsesNestedContainerLifecycleCommands(t *testing.T) {
	const inspect = `[{"Id":"wsl-id","Name":"/demo","State":{"Status":"running"},"Config":{"Image":"alpine:latest"}}]`
	runner := &fakeRunner{run: func(args []string) ([]byte, error) {
		if len(args) > 0 && args[0] == "version" {
			return []byte("wslc 2.9.3.0"), nil
		}
		if len(args) > 1 && args[1] == "create" {
			return []byte("wsl-id\n"), nil
		}
		if len(args) > 1 && args[1] == "inspect" {
			return []byte(inspect), nil
		}
		return nil, nil
	}}
	adapter := NewAdapterWithRunner(KindWSL, "wslc.exe", runner)
	sb, err := adapter.Create(context.Background(), domain.SandboxConfig{Name: "demo", Image: "alpine:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if sb.Status != domain.SandboxStatusRunning {
		t.Fatalf("unexpected WSL status: %s", sb.Status)
	}
	if err := adapter.Stop(context.Background(), sb.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(runner.calls[1], " "); got != "container create --name demo alpine:latest" {
		t.Fatalf("create argv=%q", got)
	}
	if got := strings.Join(runner.calls[len(runner.calls)-1], " "); got != "container stop wsl-id" {
		t.Fatalf("stop argv=%q", got)
	}
}

func TestCreateRejectsIncompleteConfig(t *testing.T) {
	adapter := NewAdapterWithRunner(KindApple, "container", &fakeRunner{})
	if _, err := adapter.Create(context.Background(), domain.SandboxConfig{Name: "missing-image"}); err == nil {
		t.Fatal("expected missing image error")
	}
}

func TestWSLCreateRejectsUnsupportedReadOnlyRootfs(t *testing.T) {
	adapter := NewAdapterWithRunner(KindWSL, "wslc.exe", &fakeRunner{})
	_, err := adapter.Create(context.Background(), domain.SandboxConfig{
		Name: "demo", Image: "alpine:latest", ReadOnlyRootfs: true,
	})
	if err == nil || !strings.Contains(err.Error(), "read-only rootfs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetricsAcceptsWSLArrayAndHumanReadableUnits(t *testing.T) {
	runner := &fakeRunner{run: func(args []string) ([]byte, error) {
		if len(args) > 1 && args[1] == "stats" {
			return []byte(`[{"CPUPerc":"12.5%","MemUsage":"1.5MiB / 2GiB","NetIO":"4.0kB / 8.0kB"}]`), nil
		}
		return nil, nil
	}}
	adapter := NewAdapterWithRunner(KindWSL, "wslc.exe", runner)
	metrics, err := adapter.Metrics(context.Background(), "wsl-id")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.SandboxID != "wsl-id" || metrics.CPUUsage != 12.5 || metrics.MemoryUsage != 1572864 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}

func TestInspectPreservesNativeIPAddress(t *testing.T) {
	sandbox, err := decodeSandbox([]byte(`[{"id":"apple-id","configuration":{"image":{"reference":"alpine:latest"}},"status":{"state":"running","networks":[{"ipv4Address":"192.0.2.10"}]}}]`))
	if err != nil {
		t.Fatal(err)
	}
	if sandbox.IPAddress != "192.0.2.10" {
		t.Fatalf("ip address=%q", sandbox.IPAddress)
	}
}
