// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

type lifecycleCommandRunner struct {
	responses map[string]gpu.CommandResult
}

func (runner lifecycleCommandRunner) Run(_ context.Context, name string, args ...string) (gpu.CommandResult, error) {
	key := name + " " + strings.Join(args, " ")
	if response, ok := runner.responses[key]; ok {
		return response, nil
	}
	if strings.Contains(key, "podman run --detach") {
		return gpu.CommandResult{Stdout: []byte("container-1\n")}, nil
	}
	return gpu.CommandResult{}, nil
}

func (runner lifecycleCommandRunner) RunWithEnv(_ context.Context, _ map[string]string, name string, args ...string) (gpu.CommandResult, error) {
	return runner.Run(context.Background(), name, args...)
}

func TestServiceLifecycleExecutesCreateIntentsInOrder(t *testing.T) {
	runner := lifecycleCommandRunner{responses: map[string]gpu.CommandResult{
		"podman info --format json": {Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
	}}
	action := ServiceLifecycleAction{Runner: runner}
	cpu := uint32(1000)
	mem := uint64(1073741824)
	result, err := action.Execute(context.Background(), ServiceLifecycleRequest{
		Version:        ServiceLifecycleVersion,
		SchemaVersion:  PhenoComposeLifecycleSchema,
		ManifestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		RunID:          "run",
		PodmanPipe:     "npipe:////./pipe/podman-machine-default",
		Order:          []string{"worker"},
		Intents:        []ServiceLifecycleIntent{{Phase: "create", Service: "worker", Image: "quay.io/podman/hello:latest"}},
		Services: map[string]ServiceDefinition{
			"worker": {
				Image:       "quay.io/podman/hello:latest",
				CPUMillis:   &cpu,
				MemoryBytes: &mem,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Containers["worker"] != "container-1" {
		t.Fatalf("unexpected lifecycle result: %#v", result)
	}
}

func TestServiceLifecycleRejectsUnsupportedSchema(t *testing.T) {
	_, err := (ServiceLifecycleAction{Runner: lifecycleCommandRunner{}}).Execute(context.Background(), ServiceLifecycleRequest{
		Version:        ServiceLifecycleVersion,
		SchemaVersion:  "other",
		ManifestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		RunID:          "run",
		PodmanPipe:     "npipe:////./pipe/podman-machine-default",
		Intents:        []ServiceLifecycleIntent{{Phase: "create", Service: "worker"}},
		Services:       map[string]ServiceDefinition{"worker": {Image: "img"}},
	})
	if err == nil {
		t.Fatal("expected unsupported schema to fail")
	}
}

func TestCanonicalManifestDigestRequiresLowercaseHex(t *testing.T) {
	valid := strings.Repeat("a", 64)
	if got, err := canonicalManifestDigest(valid); err != nil || got != "sha256:"+valid {
		t.Fatalf("canonicalManifestDigest(valid) = %q, %v", got, err)
	}
	for name, value := range map[string]string{
		"uppercase": strings.Repeat("A", 64),
		"non-hex":   strings.Repeat("g", 64),
		"short":     strings.Repeat("a", 63),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := canonicalManifestDigest(value); err == nil {
				t.Fatalf("canonicalManifestDigest(%q) accepted invalid digest", name)
			}
		})
	}
}
