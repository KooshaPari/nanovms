// Package ports defines the interfaces (ports) for the hexagonal architecture.
package ports

import (
	"context"
	"io"

	"github.com/kooshapari/nanovms/internal/domain"
)

// SandboxPort defines the interface for sandbox operations.
type SandboxPort interface {
	// Create creates a new sandbox
	Create(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)

	// Start starts a sandbox
	Start(ctx context.Context, id string) error

	// Stop stops a running sandbox
	Stop(ctx context.Context, id string, force bool) error

	// Delete deletes a sandbox
	Delete(ctx context.Context, id string) error

	// List lists all sandboxes
	List(ctx context.Context) ([]*domain.Sandbox, error)

	// Get gets a sandbox by ID
	Get(ctx context.Context, id string) (*domain.Sandbox, error)

	// Logs gets the logs of a sandbox
	Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error)

	// Exec executes a command in a running sandbox
	Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error)

	// Metrics gets the metrics of a sandbox
	Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error)
}

// VMFlavorPort defines the interface for VM flavor operations.
type VMFlavorPort interface {
	// Name returns the name of the VM flavor
	Name() domain.VMFlavor

	// Priority returns the priority of this flavor (higher = preferred)
	Priority() int

	// IsAvailable checks if this VM flavor is available on the current system
	IsAvailable(ctx context.Context) (bool, error)

	// CreateVM creates a new VM instance
	CreateVM(ctx context.Context, config domain.SandboxConfig) (*domain.Sandbox, error)

	// StartVM starts a VM instance
	StartVM(ctx context.Context, id string) error

	// StopVM stops a VM instance
	StopVM(ctx context.Context, id string, force bool) error

	// DeleteVM deletes a VM instance
	DeleteVM(ctx context.Context, id string) error

	// ListVMs lists all VM instances
	ListVMs(ctx context.Context) ([]*domain.Sandbox, error)

	// GetVM returns a VM by ID
	GetVM(ctx context.Context, id string) (*domain.Sandbox, error)
}

// SandboxIsolationPort defines the interface for sandbox isolation layers.
// These sit above the VM layer and provide syscall interception/filtering.
type SandboxIsolationPort interface {
	// Name returns the name of the isolation mechanism
	Name() string

	// Type returns the type of sandbox
	Type() domain.SandboxType

	// IsAvailable checks if this isolation mechanism is available
	IsAvailable(ctx context.Context) (bool, error)

	// Apply applies the isolation layer to a sandbox
	Apply(ctx context.Context, sandbox *domain.Sandbox, config domain.SandboxLayer) error

	// Remove removes the isolation layer from a sandbox
	Remove(ctx context.Context, sandbox *domain.Sandbox) error

	// Validate validates the sandbox configuration for this isolation layer
	Validate(ctx context.Context, config domain.SandboxLayer) error
}

// ImagePort defines the interface for image operations.
type ImagePort interface {
	// Pull pulls an image
	Pull(ctx context.Context, ref string) error

	// List lists all images
	List(ctx context.Context) ([]*domain.OCIImage, error)

	// Delete deletes an image
	Delete(ctx context.Context, ref string) error

	// Get gets an image by ref
	Get(ctx context.Context, ref string) (*domain.OCIImage, error)
}

// NetworkPort defines the interface for network operations.
type NetworkPort interface {
	// CreateNetwork creates a new network
	CreateNetwork(ctx context.Context, config domain.NetworkConfig) (*domain.Network, error)

	// DeleteNetwork deletes a network
	DeleteNetwork(ctx context.Context, name string) error

	// ListNetworks lists all networks
	ListNetworks(ctx context.Context) ([]*domain.Network, error)

	// Connect connects a sandbox to a network
	Connect(ctx context.Context, sandboxID, networkName string) error

	// Disconnect disconnects a sandbox from a network
	Disconnect(ctx context.Context, sandboxID, networkName string) error
}

// VolumePort defines the interface for volume operations.
type VolumePort interface {
	// CreateVolume creates a new volume
	CreateVolume(ctx context.Context, name string, sizeMB int) error

	// DeleteVolume deletes a volume
	DeleteVolume(ctx context.Context, name string) error

	// ListVolumes lists all volumes
	ListVolumes(ctx context.Context) ([]string, error)

	// InspectVolume inspects a volume
	InspectVolume(ctx context.Context, name string) (map[string]any, error)
}

// RuntimePort defines the interface for container runtime operations.
// This is the OCI runtime layer (containerd, crio, crun, etc.)
type RuntimePort interface {
	// Name returns the name of the runtime
	Name() string

	// IsAvailable checks if this runtime is available
	IsAvailable(ctx context.Context) (bool, error)

	// CreateContainer creates a new container
	CreateContainer(ctx context.Context, config domain.SandboxConfig) (string, error)

	// StartContainer starts a container
	StartContainer(ctx context.Context, id string) error

	// StopContainer stops a container
	StopContainer(ctx context.Context, id string, force bool) error

	// DeleteContainer deletes a container
	DeleteContainer(ctx context.Context, id string) error

	// ListContainers lists all containers
	ListContainers(ctx context.Context) ([]*domain.Sandbox, error)
}
