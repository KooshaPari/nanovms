// SPDX-License-Identifier: MIT OR Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseValidationCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantErr      bool
		errContains  string
		checkUnknown bool
	}{
		{
			name:        "missing required field",
			input:       configYAML("", 2, "512", ""),
			wantErr:     true,
			errContains: "config name is required",
		},
		{
			name:        "cpu zero is rejected",
			input:       configYAML("demo", 0, "512", ""),
			wantErr:     true,
			errContains: "config cpu must be between 2 and 64",
		},
		{
			name:        "cpu one is rejected",
			input:       configYAML("demo", 1, "512", ""),
			wantErr:     true,
			errContains: "config cpu must be between 2 and 64",
		},
		{
			name:        "cpu too large is rejected",
			input:       configYAML("demo", 9999, "512", ""),
			wantErr:     true,
			errContains: "config cpu must be between 2 and 64",
		},
		{
			name:        "memory garbage string is rejected",
			input:       configYAML("demo", 2, "garbage", ""),
			wantErr:     true,
			errContains: "failed to parse YAML config",
		},
		{
			name:        "memory zero string is rejected",
			input:       configYAML("demo", 2, "0MB", ""),
			wantErr:     true,
			errContains: "failed to parse YAML config",
		},
		{
			name:        "memory negative string is rejected",
			input:       configYAML("demo", 2, "-1GB", ""),
			wantErr:     true,
			errContains: "failed to parse YAML config",
		},
		{
			name:         "unknown field is allowed by current parser policy",
			input:        configYAML("demo", 2, "512", "unexpected: true"),
			checkUnknown: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := parseTestConfig(t, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse() error = nil, want error containing %q", tc.errContains)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("Parse() error = %q, want substring %q", err.Error(), tc.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error = %v", err)
			}
			if cfg == nil {
				t.Fatal("Parse() returned nil config")
			}
			if tc.checkUnknown && cfg.Name != "demo" {
				t.Fatalf("Parse() did not preserve known fields, got name %q", cfg.Name)
			}
		})
	}
}

func TestParseRoundTripValidConfig(t *testing.T) {
	t.Parallel()

	original, err := parseTestConfig(t, configYAML("demo", 2, "512", ""))
	if err != nil {
		t.Fatalf("Parse() initial error = %v", err)
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	roundTripped, err := parseTestConfig(t, string(data))
	if err != nil {
		t.Fatalf("Parse() round-trip error = %v", err)
	}

	if !reflect.DeepEqual(normalizeConfig(*original), normalizeConfig(*roundTripped)) {
		t.Fatalf("round-trip mismatch: got %+v want %+v", *roundTripped, *original)
	}
}

func parseTestConfig(t *testing.T, input string) (*NVMSConfig, error) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	return Parse(path)
}

func configYAML(name string, cpu int, memory string, extra string) string {
	base := fmt.Sprintf(`version: v1
name: %q
tier: 2
image: ghcr.io/example/demo:latest
cpu: %d
memory: %s
disk: 2048
sandbox:
  type: native
network:
  type: nat
`, name, cpu, yamlValue(memory))

	if extra == "" {
		return base
	}

	return base + extra + "\n"
}

func yamlValue(value string) string {
	if _, err := fmt.Sscanf(value, "%d", new(int)); err == nil && !strings.ContainsAny(value, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") {
		return value
	}
	return fmt.Sprintf("%q", value)
}

func normalizeConfig(cfg NVMSConfig) NVMSConfig {
	if len(cfg.Mounts) == 0 {
		cfg.Mounts = nil
	}
	if len(cfg.Env) == 0 {
		cfg.Env = nil
	}
	if len(cfg.Labels) == 0 {
		cfg.Labels = nil
	}
	if len(cfg.Sandbox.Layers) == 0 {
		cfg.Sandbox.Layers = nil
	}
	if len(cfg.Network.Ports) == 0 {
		cfg.Network.Ports = nil
	}
	return cfg
}
