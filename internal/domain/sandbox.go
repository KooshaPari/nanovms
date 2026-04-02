// Package domain contains the core business logic and entities.
package domain

import (
	"fmt"
	"time"
)

// RuntimeType represents the type of container runtime.
type RuntimeType string

const (
	RuntimeDocker    RuntimeType = "docker"
	RuntimeContainerd RuntimeType = "containerd"
	RuntimeCrun      RuntimeType = "crun"
	RuntimeGvisor    RuntimeType = "gvisor"
	RuntimeWasmtime  RuntimeType = "wasmtime"
	RuntimeLima      RuntimeType = "lima"
	RuntimeWSL       RuntimeType = "wsl"
	RuntimeNative    RuntimeType = "native"
)

// SandboxConfig represents the configuration for a sandboxed environment.
type SandboxConfig struct {
	Name        string            `yaml:"name"`
	Runtime     RuntimeType       `yaml:"runtime"`
	Image       string            `yaml:"image"`
	Mounts      []Mount           `yaml:"mounts"`
	Environment map[string]string `yaml:"environment"`
	Networking  bool              `yaml:"networking"`
	Privileged  bool              `yaml:"privileged"`
	MemoryMB    int               `yaml:"memory_mb"`
	CPUCount    int               `yaml:"cpu_count"`
}

// Mount represents a volume mount.
type Mount struct {
	Source      string `yaml:"source"`
	Target      string `yaml:"target"`
	ReadOnly    bool   `yaml:"read_only"`
	BindOptions string `yaml:"bind_options"`
}

// Sandbox represents a running sandbox instance.
type Sandbox struct {
	ID        string       `yaml:"id"`
	Name      string       `yaml:"name"`
	Runtime   RuntimeType  `yaml:"runtime"`
	Status    SandboxStatus `yaml:"status"`
	CreatedAt time.Time   `yaml:"created_at"`
	Config    SandboxConfig `yaml:"config"`
}

// SandboxStatus represents the current status of a sandbox.
type SandboxStatus string

const (
	StatusCreated   SandboxStatus = "created"
	StatusRunning   SandboxStatus = "running"
	StatusPaused    SandboxStatus = "paused"
	StatusStopped   SandboxStatus = "stopped"
	StatusFailed    SandboxStatus = "failed"
)

// OCIImage represents an OCI-compliant container image.
type OCIImage struct {
	Registry string `yaml:"registry"`
	Name    string `yaml:"name"`
	Tag     string `yaml:"tag"`
	Digest  string `yaml:"digest"`
}

// ParseImage parses an image string into an OCIImage.
func ParseImage(image string) (*OCIImage, error) {
	// Handle registry/name:tag format
	var registry, name, tag, digest string

	// Simple parsing - can be enhanced for full OCI spec
	parts := splitImageString(image)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid image format: %s", image)
	}

	if len(parts) >= 1 {
		name = parts[0]
	}
	if len(parts) >= 2 {
		tag = parts[1]
	}
	if len(parts) >= 3 {
		digest = parts[2]
	}

	return &OCIImage{
		Registry: registry,
		Name:    name,
		Tag:     tag,
		Digest:  digest,
	}, nil
}

func splitImageString(s string) []string {
	// Simplified - just split on :
	var parts []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, s[i])
		}
	}
	if current != nil {
		parts = append(parts, string(current))
	}
	return parts
}
