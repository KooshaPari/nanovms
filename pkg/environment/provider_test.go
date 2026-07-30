// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

type fakeInventory struct {
	devices []gpu.Device
	err     error
}

func (fake *fakeInventory) Inventory(context.Context) ([]gpu.Device, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	return append([]gpu.Device(nil), fake.devices...), nil
}

type toolkitCommandResponse struct {
	result gpu.CommandResult
	err    error
}

type toolkitCommandRunner struct {
	responses map[string]toolkitCommandResponse
	commands  [][]string
}

func (runner *toolkitCommandRunner) Run(_ context.Context, name string, args ...string) (gpu.CommandResult, error) {
	command := append([]string{name}, args...)
	runner.commands = append(runner.commands, command)
	key := name
	if name == "wsl.exe" && len(args) >= 4 {
		key = strings.Join(append([]string{name}, args...), " ")
	}
	response, ok := runner.responses[key]
	if !ok {
		response, ok = runner.responses[name]
	}
	if !ok {
		return gpu.CommandResult{}, errors.New("unexpected command: " + key)
	}
	return response.result, response.err
}

func ampereDevice() gpu.Device {
	return gpu.Device{
		UUID:              AmpereGPUUUID,
		Name:              "NVIDIA GeForce RTX 3090 Ti",
		Architecture:      "Ampere",
		ComputeCapability: "8.6",
		DriverVersion:     "580.1",
		DriverCUDACeiling: "13.0",
		Observations:      []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 1}},
	}
}

func pascalDevice() gpu.Device {
	return gpu.Device{
		UUID:              PascalGPUUUID,
		Name:              "NVIDIA GeForce GTX 1080 Ti",
		Architecture:      "Pascal",
		ComputeCapability: "6.1",
		DriverVersion:     "580.1",
		DriverCUDACeiling: "13.0",
		Observations:      []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 0}},
	}
}

func testProvider(t *testing.T, runner *toolkitCommandRunner, devices []gpu.Device) Provider {
	t.Helper()
	return Provider{
		Inventory: &fakeInventory{devices: devices},
		Toolkit: ToolkitResolver{
			Inspector: HostToolkitInspector{Runner: runner},
		},
		State: &MemoryStateStore{},
	}
}

func validRequest(profile ProfileID) Request {
	return Request{Version: ProviderVersion, Profile: profile}
}

func TestAmperePlanSuccess(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.1")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector.ToolkitRoot = root

	result, err := provider.Plan(context.Background(), validRequest(ProfileAmpere))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Contract.Profile != ProfileAmpere || result.Contract.Digest == "" {
		t.Fatalf("plan result = %#v", result)
	}
	if result.Contract.Toolkit.Observed != "13.0" || result.Contract.Compatibility.Compatible != true {
		t.Fatalf("toolkit/compatibility = %#v", result.Contract)
	}
	if len(result.Contract.Mutations) != 3 {
		t.Fatalf("mutations = %#v", result.Contract.Mutations)
	}
}

func TestPascalCUDA13Reject(t *testing.T) {
	device := pascalDevice()
	err := gpu.ValidateCompatibility(device, gpu.ArtifactRequirements{CUDAToolkit: "13.0", CompiledKernels: true})
	if err == nil {
		t.Fatal("expected Pascal CUDA 13 rejection")
	}

	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{device})
	provider.Toolkit.Inspector.ToolkitRoot = root

	_, err = provider.Plan(context.Background(), validRequest(ProfilePascal))
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Code != "toolkit_version_mismatch" && !strings.Contains(providerErr.Err.Error(), "incompatible with Pascal") {
		t.Fatalf("error = %v, want Pascal CUDA 13 rejection", err)
	}
}

func TestPascalCUDA12Accept(t *testing.T) {
	tests := []struct {
		name    string
		release string
	}{
		{name: "target 12.9", release: "12.9"},
		{name: "transitional 12.4", release: "12.4"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "cuda", "v"+test.release)
			nvcc := filepath.Join(root, "bin", "nvcc.exe")
			runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
				nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release " + test.release + ", V" + test.release + ".0")}},
			}}
			provider := testProvider(t, runner, []gpu.Device{pascalDevice()})
			provider.Toolkit.Inspector.ToolkitRoot = root

			result, err := provider.Plan(context.Background(), validRequest(ProfilePascal))
			if err != nil {
				t.Fatal(err)
			}
			if !result.Success || result.Contract.Toolkit.Observed != test.release {
				t.Fatalf("plan result = %#v, err = %v", result, err)
			}
		})
	}
}

func TestApplyIdempotentReapply(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector.ToolkitRoot = root
	request := validRequest(ProfileAmpere)

	first, err := provider.Apply(context.Background(), request)
	if err != nil || !first.Success || first.NoOp {
		t.Fatalf("first apply = %#v, err = %v", first, err)
	}
	second, err := provider.Apply(context.Background(), request)
	if err != nil || !second.Success || !second.NoOp {
		t.Fatalf("second apply = %#v, err = %v", second, err)
	}
	if first.Contract.Digest != second.Contract.Digest {
		t.Fatalf("digest drift: %q vs %q", first.Contract.Digest, second.Contract.Digest)
	}
}

func TestWSLMetadataDigestStability(t *testing.T) {
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		"wsl.exe -d Ubuntu-24.04 -- uname -r": {result: gpu.CommandResult{Stdout: []byte("5.15.167.4-microsoft-standard-WSL2\n")}},
	}}
	first, err := InspectWSL(context.Background(), runner, "Ubuntu-24.04")
	if err != nil {
		t.Fatal(err)
	}
	second, err := InspectWSL(context.Background(), runner, "Ubuntu-24.04")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == "" || first.Digest != second.Digest {
		t.Fatalf("digest = %q vs %q", first.Digest, second.Digest)
	}
	if first.Kernel != "5.15.167.4-microsoft-standard-WSL2" {
		t.Fatalf("kernel = %q", first.Kernel)
	}
}

func TestToolkitDiscoveryPrecedence(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows toolkit precedence contract")
	}
	configuredRoot := filepath.Join(t.TempDir(), "configured")
	versionEnvRoot := filepath.Join(t.TempDir(), "cuda-path-v13")
	cudaPathRoot := filepath.Join(t.TempDir(), "cuda-path")
	configuredNVCC := filepath.Join(configuredRoot, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		configuredNVCC: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.1")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector = HostToolkitInspector{
		Runner:      runner,
		ToolkitRoot: configuredRoot,
		LookupEnv: func(key string) (string, bool) {
			values := map[string]string{
				"CUDA_PATH_V13_0": versionEnvRoot,
				"CUDA_PATH":       cudaPathRoot,
			}
			value, ok := values[key]
			return value, ok
		},
		ProgramFilesRoots: []string{`C:\Program Files`},
	}

	result, err := provider.Plan(context.Background(), validRequest(ProfileAmpere))
	if err != nil {
		t.Fatal(err)
	}
	if result.Contract.Toolkit.Root != configuredRoot {
		t.Fatalf("resolved toolkit root = %q, want configured root", result.Contract.Toolkit.Root)
	}
	if len(runner.commands) != 1 || runner.commands[0][0] != configuredNVCC {
		t.Fatalf("discovery did not stop at configured root: %#v", runner.commands)
	}
}

func TestEmptyInventoryFailClosed(t *testing.T) {
	provider := Provider{
		Inventory: &fakeInventory{devices: nil},
		Toolkit: ToolkitResolver{
			Inspector: HostToolkitInspector{Runner: &toolkitCommandRunner{}},
		},
	}
	_, err := provider.Plan(context.Background(), validRequest(ProfileAmpere))
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != "inventory_empty" {
		t.Fatalf("error = %v, want inventory_empty", err)
	}
}

func TestContractDigestDeterministic(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector.ToolkitRoot = root

	first, err := provider.Plan(context.Background(), validRequest(ProfileAmpere))
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.Plan(context.Background(), validRequest(ProfileAmpere))
	if err != nil {
		t.Fatal(err)
	}
	if first.Contract.Digest != second.Contract.Digest {
		t.Fatalf("digest drift: %q vs %q", first.Contract.Digest, second.Contract.Digest)
	}
	payload, err := first.Contract.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"digest"`) {
		t.Fatalf("canonical JSON must exclude digest field: %s", payload)
	}
}

func TestVerifyRequiresAppliedState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector.ToolkitRoot = root

	_, err := provider.Verify(context.Background(), validRequest(ProfileAmpere))
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != "environment_drift" {
		t.Fatalf("error = %v, want environment_drift", err)
	}
}

func TestPlanRecordsWSLMetadata(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		nvcc:                                  {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
		"wsl.exe -d Ubuntu-24.04 -- uname -r": {result: gpu.CommandResult{Stdout: []byte("5.15.167.4-microsoft-standard-WSL2\n")}},
	}}
	provider := testProvider(t, runner, []gpu.Device{ampereDevice()})
	provider.Toolkit.Inspector.ToolkitRoot = root
	request := validRequest(ProfileAmpere)
	request.WSLDistribution = "Ubuntu-24.04"

	result, err := provider.Plan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Contract.WSL == nil || result.Contract.WSL.Distribution != "Ubuntu-24.04" || result.Contract.WSL.Digest == "" {
		t.Fatalf("WSL metadata = %#v", result.Contract.WSL)
	}
	want := []string{"wsl.exe", "-d", "Ubuntu-24.04", "--", "uname", "-r"}
	if len(runner.commands) != 2 || !reflect.DeepEqual(runner.commands[1], want) {
		t.Fatalf("WSL probe = %#v, want %#v", runner.commands[1:], want)
	}
}
