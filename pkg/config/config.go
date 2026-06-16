// SPDX-License-Identifier: MIT OR Apache-2.0
// Package config provides the NVMS configuration parser with YAML support.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/kooshapari/nanovms/internal/domain"
)

// NVMSConfig represents the top-level NVMS configuration file.
type NVMSConfig struct {
	Version string            `yaml:"version" json:"version"`
	Name    string            `yaml:"name" json:"name"`
	Tier    int               `yaml:"tier" json:"tier"`
	Image   string            `yaml:"image" json:"image"`
	CPU     int               `yaml:"cpu" json:"cpu"`
	Memory  int               `yaml:"memory" json:"memory"`
	Disk    int               `yaml:"disk" json:"disk"`
	Sandbox SandboxConfig     `yaml:"sandbox" json:"sandbox"`
	Network NetworkConfig     `yaml:"network" json:"network"`
	Mounts  []MountConfig     `yaml:"mounts" json:"mounts"`
	Env     map[string]string `yaml:"env" json:"env"`
	Labels  map[string]string `yaml:"labels" json:"labels"`
}

// SandboxConfig holds sandbox-specific configuration.
type SandboxConfig struct {
	Type     string   `yaml:"type" json:"type"`
	Layer    string   `yaml:"layer" json:"layer"`
	Layers   []string `yaml:"layers" json:"layers"`
	ReadOnly bool     `yaml:"read_only" json:"read_only"`
	Seccomp  string   `yaml:"seccomp" json:"seccomp"`
	Firejail string   `yaml:"firejail" json:"firejail"`
}

// NetworkConfig holds network configuration.
type NetworkConfig struct {
	Type   string   `yaml:"type" json:"type"`
	Subnet string   `yaml:"subnet" json:"subnet"`
	Ports  []string `yaml:"ports" json:"ports"`
}

// MountConfig holds mount configuration.
type MountConfig struct {
	Source   string `yaml:"source" json:"source"`
	Target   string `yaml:"target" json:"target"`
	ReadOnly bool   `yaml:"read_only" json:"read_only"`
	Type     string `yaml:"type" json:"type"`
}

// Parse reads and parses an NVMS YAML configuration file.
func Parse(path string) (*NVMSConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg NVMSConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate performs basic validation on the parsed configuration.
func (c *NVMSConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("config name is required")
	}
	if c.Image == "" {
		return fmt.Errorf("config image is required")
	}
	if c.Tier < 1 || c.Tier > 3 {
		return fmt.Errorf("config tier must be 1, 2, or 3")
	}
	if c.CPU < 2 || c.CPU > 64 {
		return fmt.Errorf("config cpu must be between 2 and 64")
	}
	if c.Memory < 64 {
		return fmt.Errorf("config memory must be at least 64 MB")
	}
	return nil
}

// ToDomainConfig converts the NVMS config to a domain SandboxConfig.
func (c *NVMSConfig) ToDomainConfig() domain.SandboxConfig {
	mounts := make([]domain.Mount, 0, len(c.Mounts))
	for _, m := range c.Mounts {
		mounts = append(mounts, domain.Mount{
			Source:   m.Source,
			Target:   m.Target,
			Type:     m.Type,
			ReadOnly: m.ReadOnly,
		})
	}

	var vmFlavor domain.VMFlavor
	switch c.Tier {
	case 1:
		vmFlavor = domain.VMFlavorWasm
	case 2:
		vmFlavor = domain.VMFlavorLima
	case 3:
		vmFlavor = domain.VMFlavorMicroVM
	default:
		vmFlavor = domain.VMFlavorNative
	}

	return domain.SandboxConfig{
		Name:            c.Name,
		Image:           c.Image,
		VMType:          vmFlavor,
		SandboxType:     domain.SandboxType(c.Sandbox.Type),
		Mounts:          mounts,
		Environment:     c.Env,
		Labels:          c.Labels,
		ReadOnlyRootfs:  c.Sandbox.ReadOnly,
		SeccompProfile:  c.Sandbox.Seccomp,
		FirejailProfile: c.Sandbox.Firejail,
	}
}
