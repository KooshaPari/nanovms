// Package ports defines the interfaces (ports) for the hexagonal architecture.
package ports

import (
	"context"
	"io"

	"github.com/kooshapari/devenv-abstraction/internal/domain"
)

// RuntimePort defines the interface for container runtime adapters.
type RuntimePort interface {
	// Name returns the runtime name
	Name() string
	// Create creates a new sandboxed environment
	Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)
	// Start starts a sandbox
	Start(ctx context.Context, id string) error
	// Stop stops a sandbox
	Stop(ctx context.Context, id string) error
	// Delete deletes a sandbox
	Delete(ctx context.Context, id string) error
	// List lists all sandboxes
	List(ctx context.Context) ([]domain.Sandbox, error)
	// Status returns the status of a sandbox
	Status(ctx context.Context, id string) (domain.SandboxStatus, error)
	// Exec executes a command in a sandbox
	Exec(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error
	// Pull pulls an image
	Pull(ctx context.Context, image string) error
}

// ImagePort defines the interface for image registry operations.
type ImagePort interface {
	// Pull pulls an image from the registry
	Pull(ctx context.Context, image string) error
	// ListImages lists available images
	ListImages(ctx context.Context) ([]domain.OCIImage, error)
	// Delete removes an image
	Delete(ctx context.Context, image string) error
}

// FilesystemPort defines the interface for filesystem operations.
type FilesystemPort interface {
	// Mount mounts a directory into the sandbox
	Mount(ctx context.Context, sandboxID, source, target string, readOnly bool) error
	// Unmount unmounts a directory
	Unmount(ctx context.Context, sandboxID, target string) error
	// ListMounts lists all mounts for a sandbox
	ListMounts(ctx context.Context, sandboxID string) ([]domain.Mount, error)
}

// NetworkPort defines the interface for network operations.
type NetworkPort interface {
	// CreateNetwork creates a new network
	CreateNetwork(ctx context.Context, name string, subnet string) error
	// DeleteNetwork deletes a network
	DeleteNetwork(ctx context.Context, name string) error
	// Connect connects a sandbox to a network
	Connect(ctx context.Context, sandboxID, network string) error
	// Disconnect disconnects a sandbox from a network
	Disconnect(ctx context.Context, sandboxID, network string) error
}
