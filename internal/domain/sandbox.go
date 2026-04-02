// Package domain contains the core domain models.
package domain

import (
	"context"
	"fmt"
	"time"
)

// SandboxType represents the type of sandbox isolation.
type SandboxType string

const (
	// SandboxTypeVM indicates a full virtual machine
	SandboxTypeVM SandboxType = "vm"
	// SandboxTypeContainer indicates container-based isolation
	SandboxTypeContainer SandboxType = "container"
	// SandboxTypeWasm indicates WebAssembly-based isolation
	SandboxTypeWasm SandboxType = "wasm"
	// SandboxTypeProcess indicates process-level isolation (gVisor, landlock)
	SandboxTypeProcess SandboxType = "process"
	// SandboxTypeNative indicates native OS-level isolation (bwrap, firejail, namespaces)
	SandboxTypeNative SandboxType = "native"
)

// NativeSandboxType represents the specific native sandbox implementation.
type NativeSandboxType string

const (
	// NativeSandboxBwrap bubblewrap/bwrap - Linux namespace isolation
	NativeSandboxBwrap NativeSandboxType = "bwrap"
	// NativeSandboxFirejail - Firejail sandbox
	NativeSandboxFirejail NativeSandboxType = "firejail"
	// NativeSandboxUnshare - Linux unshare namespace isolation
	NativeSandboxUnshare NativeSandboxType = "unshare"
	// NativeSandboxChroot - chroot isolation
	NativeSandboxChroot NativeSandboxType = "chroot"
	// NativeSandboxWindowsContainer - Windows container isolation (process namespace)
	NativeSandboxWindowsContainer NativeSandboxType = "windows-container"
	// NativeSandboxMacOSContain - macOS sandbox-exec isolation
	NativeSandboxMacOSContain NativeSandboxType = "sandbox-exec"
)

// VMFlavor represents the VM implementation flavor per OS.
type VMFlavor string

const (
	// VMFlavorNative uses native hypervisor (HyperKit, Hyper-V, KVM)
	VMFlavorNative VMFlavor = "native"
	// VMFlavorLima uses Lima with vz driver on Mac, native on Linux
	VMFlavorLima VMFlavor = "lima"
	// VMFlavorWSL uses Windows Subsystem for Linux
	VMFlavorWSL VMFlavor = "wsl"
	// VMFlavorMicroVM uses Firecracker microVM
	VMFlavorMicroVM VMFlavor = "microvm"
	// VMFlavorWasm uses WebAssembly runtime
	VMFlavorWasm VMFlavor = "wasm"
)

// SandboxLayer represents a sandbox isolation layer.
// It can be stacked - e.g., Native (bwrap) -> Process (gVisor) -> VM (Firecracker)
type SandboxLayer struct {
	// Type of sandbox isolation
	Type SandboxType `json:"type"`
	// Implementation name (e.g., "gvisor", "bwrap", "firejail")
	Implementation string `json:"implementation"`
	// Config is the layer-specific configuration
	Config interface{} `json:"config,omitempty"`
}

// SandboxConfig holds sandbox configuration.
type SandboxConfig struct {
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	VMType         VMType           `json:"vm_type"`
	VMConfig       *VMConfig        `json:"vm_config,omitempty"`
	SandboxType    SandboxType      `json:"sandbox_type"`
	SandboxLayers  []SandboxLayer   `json:"sandbox_layers,omitempty"`
	NativeSandbox  *NativeSandboxConfig `json:"native_sandbox,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// ResourceConfig defines CPU/memory limits.
type ResourceConfig struct {
	CPU      int    `json:"cpu"`      // Number of CPUs
	MemoryMB int    `json:"memory"`  // Memory in MB
	DiskMB   int    `json:"disk"`    // Disk in MB
}

// NetworkConfig defines network configuration.
type NetworkConfig struct {
	Type      string `json:"type"`       // bridge, nat, none
	Subnet    string `json:"subnet"`    // Subnet CIDR
	Ports     []PortMapping `json:"ports"`
}

// PortMapping defines port forwarding.
type PortMapping struct {
	HostPort     int    `json:"host_port"`
	ContainerPort int   `json:"container_port"`
	Protocol     string `json:"protocol"` // tcp, udp
}

// FilesystemMount defines a filesystem mount.
type FilesystemMount struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	ReadOnly    bool   `json:"read_only"`
	Type        string `json:"type"` // bind, volume
}

// Sandbox represents a running sandbox.
type Sandbox struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Status      SandboxStatus     `json:"status"`
	VMFlavor    VMFlavor        `json:"vm_flavor"`
	Layers      []SandboxLayer   `json:"layers"`       // Active isolation layers
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at"`
	IPAddress   string           `json:"ip_address"`
	Ports      []PortMapping    `json:"ports"`
	Mounts     []Mount          `json:"mounts"`
	Metrics    *SandboxMetrics  `json:"metrics,omitempty"`
}

// SandboxStatus represents the status of a sandbox.
type SandboxStatus string

const (
	SandboxStatusPending   SandboxStatus = "pending"
	SandboxStatusRunning   SandboxStatus = "running"
	SandboxStatusStopped   SandboxStatus = "stopped"
	SandboxStatusFailed    SandboxStatus = "failed"
	SandboxStatusDeleting  SandboxStatus = "deleting"
)

// Mount represents a filesystem mount.
type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// SandboxMetrics contains usage metrics for a sandbox.
type SandboxMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`    // Percentage
	MemoryUsage int64   `json:"memory_usage"` // Bytes
	DiskUsage   int64   `json:"disk_usage"`  // Bytes
	NetworkRx   int64   `json:"network_rx"`  // Bytes received
	NetworkTx   int64   `json:"network_tx"`  // Bytes sent
}

// String returns a string representation of the sandbox.
func (s *Sandbox) String() string {
	return fmt.Sprintf("Sandbox(%s, %s, %s)", s.ID, s.Name, s.Status)
}

// IsRunning returns true if the sandbox is running.
func (s *Sandbox) IsRunning() bool {
	return s.Status == SandboxStatusRunning
}

// OCIImage represents an OCI container image.
type OCIImage struct {
	Ref   string `json:"ref"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
	Created time.Time `json:"created"`
	Platform PlatformInfo `json:"platform"`
}

// PlatformInfo describes the image platform.
type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant,omitempty"`
}

// Network represents a virtual network.
type Network struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
	Type   string `json:"type"` // bridge, nat
	ID     string `json:"id"`
}

// VMAdapter defines the interface for VM implementations.
type VMAdapter interface {
	// CreateVM creates a new VM with the given config.
	CreateVM(ctx context.Context, config *VMConfig) (*VMinstance, error)
	// StartVM starts an existing VM.
	StartVM(ctx context.Context, id string) error
	// StopVM stops a running VM.
	StopVM(ctx context.Context, id string) error
	// DeleteVM deletes a VM.
	DeleteVM(ctx context.Context, id string) error
	// ListVMs lists all VMs.
	ListVMs(ctx context.Context) ([]*VMinstance, error)
	// GetVM gets a VM by ID.
	GetVM(ctx context.Context, id string) (*VMinstance, error)
}

// VMConfig contains configuration for creating a VM.
type VMConfig struct {
	Name      string
	VMFlavor  VMFlavor
	Image     string
	Resources ResourceConfig
	Network   NetworkConfig
}

// VMInstance represents a running VM instance.
type VMInstance struct {
	ID        string
	Name      string
	Flavor    VMFlavor
	Status    string
	IPAddress string
	Ports     []PortMapping
}

// Ensure VMAdapter implementations satisfy the interface.
var _ VMAdapter = (*VMAdapter)(nil)
