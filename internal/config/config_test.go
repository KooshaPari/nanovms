package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nvms.toml")
	contents := strings.Join([]string{
		"name = \"demo\"",
		"image = \"ghcr.io/example/demo:latest\"",
		"tier = 2",
		"cpu = 2",
		"memory = 256",
	}, "\n")

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Name != "demo" {
		t.Fatalf("Name = %q, want demo", cfg.Name)
	}
	if cfg.Image != "ghcr.io/example/demo:latest" {
		t.Fatalf("Image = %q, want image value", cfg.Image)
	}
	if cfg.Tier != 2 {
		t.Fatalf("Tier = %d, want 2", cfg.Tier)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.toml")
	contents := strings.Join([]string{
		"name = \"\"",
		"image = \"\"",
		"tier = 0",
		"cpu = 0",
		"memory = 32",
	}, "\n")

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}

	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("Load() error = %v, want name validation", err)
	}
}

func TestPathFromArgs(t *testing.T) {
	t.Parallel()

	path, err := pathFromArgs([]string{"nvms", "nvms.toml"})
	if err != nil {
		t.Fatalf("pathFromArgs() error = %v", err)
	}
	if path != "nvms.toml" {
		t.Fatalf("pathFromArgs() = %q, want nvms.toml", path)
	}

	_, err = pathFromArgs([]string{"nvms"})
	if err == nil {
		t.Fatal("pathFromArgs() error = nil, want usage error")
	}
}
