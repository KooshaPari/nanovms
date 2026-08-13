package orchestrate

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

type fakeSandboxPort struct {
	startErr    error
	createCalls int
	startCalls  int
	deleteCalls int
	getCalls    int
}

func (p *fakeSandboxPort) Create(context.Context, domain.SandboxConfig) (*domain.Sandbox, error) {
	p.createCalls++
	return &domain.Sandbox{ID: "sandbox-1", Status: domain.SandboxStatusPending}, nil
}

func (p *fakeSandboxPort) Start(context.Context, string) error {
	p.startCalls++
	return p.startErr
}

func (p *fakeSandboxPort) Stop(context.Context, string, bool) error { return nil }

func (p *fakeSandboxPort) Delete(context.Context, string) error {
	p.deleteCalls++
	return nil
}

func (p *fakeSandboxPort) List(context.Context) ([]*domain.Sandbox, error) { return nil, nil }

func (p *fakeSandboxPort) Get(context.Context, string) (*domain.Sandbox, error) {
	p.getCalls++
	return &domain.Sandbox{ID: "sandbox-1", Status: domain.SandboxStatusRunning}, nil
}

func (p *fakeSandboxPort) Logs(context.Context, string, bool) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (p *fakeSandboxPort) Exec(context.Context, string, []string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (p *fakeSandboxPort) Metrics(context.Context, string) (*domain.SandboxMetrics, error) {
	return &domain.SandboxMetrics{SandboxID: "sandbox-1"}, nil
}

func TestSandboxPortDispatcherCreatesStartsAndRefreshes(t *testing.T) {
	port := &fakeSandboxPort{}
	sandbox, err := newSandboxPortDispatcher(port).Deploy(context.Background(), domain.SandboxConfig{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if sandbox.Status != domain.SandboxStatusRunning || port.createCalls != 1 || port.startCalls != 1 || port.getCalls != 1 {
		t.Fatalf("sandbox=%#v port=%#v", sandbox, port)
	}
}

func TestSandboxPortDispatcherRollsBackFailedStart(t *testing.T) {
	port := &fakeSandboxPort{startErr: errors.New("engine unavailable")}
	_, err := newSandboxPortDispatcher(port).Deploy(context.Background(), domain.SandboxConfig{Name: "demo"})
	if err == nil || port.createCalls != 1 || port.startCalls != 1 || port.deleteCalls != 1 {
		t.Fatalf("err=%v port=%#v", err, port)
	}
}

func TestNewEngineRegistersProviderNativeDispatchers(t *testing.T) {
	engine := NewEngine()
	for _, backend := range []nvmsruntime.BackendID{
		nvmsruntime.BackendPodman,
		nvmsruntime.BackendAppleContainers,
		nvmsruntime.BackendWSLContainers,
	} {
		if engine.backendDispatchers[backend] == nil {
			t.Fatalf("missing dispatcher for %s", backend)
		}
	}
}
