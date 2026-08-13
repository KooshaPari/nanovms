package orchestrate

import (
	"context"
	"fmt"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// sandboxPortDispatcher adapts the lifecycle port to the composition
// dispatcher contract. Create and Start are deliberately separate so a
// failed Start can remove the stopped resource before returning.
type sandboxPortDispatcher struct {
	port ports.SandboxPort
}

func newSandboxPortDispatcher(port ports.SandboxPort) BackendDispatcher {
	return &sandboxPortDispatcher{port: port}
}

func (d *sandboxPortDispatcher) Deploy(ctx context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	if d == nil || d.port == nil {
		return nil, fmt.Errorf("sandbox provider is not configured")
	}
	sandbox, err := d.port.Create(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if sandbox == nil || sandbox.ID == "" {
		return nil, fmt.Errorf("sandbox provider returned no identity")
	}
	if err := d.port.Start(ctx, sandbox.ID); err != nil {
		if cleanupErr := d.port.Delete(context.WithoutCancel(ctx), sandbox.ID); cleanupErr != nil {
			return nil, fmt.Errorf("start failed: %w (cleanup failed: %v)", err, cleanupErr)
		}
		return nil, fmt.Errorf("start failed: %w", err)
	}
	if refreshed, err := d.port.Get(ctx, sandbox.ID); err == nil && refreshed != nil {
		return refreshed, nil
	}
	sandbox.Status = domain.SandboxStatusRunning
	return sandbox, nil
}
