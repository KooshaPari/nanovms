// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

const testUUID gpu.UUID = "GPU-123e4567-e89b-12d3-a456-426614174000"

type fakeEvaluationInspector struct {
	inspection EvaluationInspection
	err        error
	calls      atomic.Int32
}

func (fake *fakeEvaluationInspector) Inspect(context.Context, EvaluationRequest) (EvaluationInspection, error) {
	fake.calls.Add(1)
	return fake.inspection, fake.err
}

type fakeInventoryProvider struct {
	devices []gpu.Device
	err     error
	calls   atomic.Int32
}

func (fake *fakeInventoryProvider) Inventory(context.Context) ([]gpu.Device, error) {
	fake.calls.Add(1)
	return append([]gpu.Device(nil), fake.devices...), fake.err
}

type fakeEvaluationRunner struct {
	mu          sync.Mutex
	request     *EvaluationRequest
	result      gpu.CommandResult
	err         error
	command     string
	arguments   []string
	environment map[string]string
	active      int
	maxActive   int
	sequence    int
	delay       time.Duration
}

func (fake *fakeEvaluationRunner) RunWithEnv(_ context.Context, environment map[string]string, command string, arguments ...string) (gpu.CommandResult, error) {
	fake.mu.Lock()
	fake.command = command
	fake.arguments = append([]string(nil), arguments...)
	fake.environment = cloneEnvironment(environment)
	fake.active++
	if fake.active > fake.maxActive {
		fake.maxActive = fake.active
	}
	fake.sequence++
	sequence := fake.sequence
	request := fake.request
	fake.mu.Unlock()

	if fake.delay > 0 {
		time.Sleep(fake.delay)
	}
	if request != nil {
		job := filepath.Join(request.OutputRoot, fmt.Sprintf("job-%d", sequence))
		if err := os.Mkdir(job, 0o700); err != nil {
			return gpu.CommandResult{}, err
		}
		data, _ := json.Marshal(map[string]any{"invocation": request.LockInvocation})
		if err := os.WriteFile(filepath.Join(job, JobLockFilename), data, 0o600); err != nil {
			return gpu.CommandResult{}, err
		}
	}

	fake.mu.Lock()
	fake.active--
	fake.mu.Unlock()
	return fake.result, fake.err
}

type fakeReservations struct {
	mu         sync.Mutex
	reserves   int
	releases   int
	releaseErr error
}

type injectedEvaluationFilesystem struct {
	osEvaluationFilesystem
	mkdirErr      error
	openErr       error
	removeRootErr error
	root          string
	available     uint64
	availableErr  error
	onAvailable   func(string)
	availableOnce sync.Once
}

func (filesystem *injectedEvaluationFilesystem) Mkdir(path string, mode os.FileMode) error {
	if filesystem.mkdirErr != nil && path == filesystem.root {
		return filesystem.mkdirErr
	}
	return filesystem.osEvaluationFilesystem.Mkdir(path, mode)
}

func (filesystem *injectedEvaluationFilesystem) OpenFile(path string, flag int, mode os.FileMode) (*os.File, error) {
	if filesystem.openErr != nil {
		return nil, filesystem.openErr
	}
	return filesystem.osEvaluationFilesystem.OpenFile(path, flag, mode)
}

func (filesystem *injectedEvaluationFilesystem) Remove(path string) error {
	if filesystem.removeRootErr != nil && path == filesystem.root {
		return filesystem.removeRootErr
	}
	return filesystem.osEvaluationFilesystem.Remove(path)
}

func (filesystem *injectedEvaluationFilesystem) AvailableSpace(path string) (uint64, error) {
	filesystem.availableOnce.Do(func() {
		if filesystem.onAvailable != nil {
			filesystem.onAvailable(path)
		}
	})
	return filesystem.available, filesystem.availableErr
}

type sequentialCommandRunner struct {
	responses    []gpu.CommandResult
	commands     [][]string
	environments []map[string]string
}

func (fake *sequentialCommandRunner) Run(_ context.Context, command string, arguments ...string) (gpu.CommandResult, error) {
	fake.commands = append(fake.commands, append([]string{command}, arguments...))
	if len(fake.responses) == 0 {
		return gpu.CommandResult{}, errors.New("unexpected inspection command")
	}
	result := fake.responses[0]
	fake.responses = fake.responses[1:]
	return result, nil
}

func (fake *sequentialCommandRunner) RunWithEnv(_ context.Context, environment map[string]string, command string, arguments ...string) (gpu.CommandResult, error) {
	fake.environments = append(fake.environments, cloneEnvironment(environment))
	return fake.Run(context.Background(), command, arguments...)
}

type toolkitCommandResponse struct {
	result gpu.CommandResult
	err    error
}

type toolkitCommandRunner struct {
	responses map[string]toolkitCommandResponse
	commands  [][]string
}

func (fake *toolkitCommandRunner) Run(_ context.Context, command string, arguments ...string) (gpu.CommandResult, error) {
	fake.commands = append(fake.commands, append([]string{command}, arguments...))
	response, ok := fake.responses[command]
	if !ok {
		return gpu.CommandResult{}, errors.New("executable not found")
	}
	return response.result, response.err
}

func (fake *toolkitCommandRunner) RunWithEnv(_ context.Context, _ map[string]string, command string, arguments ...string) (gpu.CommandResult, error) {
	return fake.Run(context.Background(), command, arguments...)
}

type fakeWSLToolkitProvider struct {
	roots        []string
	err          error
	distribution string
	version      string
}

func (fake *fakeWSLToolkitProvider) ToolkitRoots(_ context.Context, distribution, version string) ([]string, error) {
	fake.distribution = distribution
	fake.version = version
	return append([]string(nil), fake.roots...), fake.err
}

func (fake *fakeReservations) Reserve(_ context.Context, uuids []gpu.UUID, owner string, ttl time.Duration) (gpu.ReservationLease, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.reserves++
	return gpu.ReservationLease{Token: "token", Owner: owner, UUIDs: uuids, ExpiresAt: time.Now().Add(ttl)}, nil
}

func (fake *fakeReservations) Release(context.Context, gpu.ReservationLease) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.releases++
	return fake.releaseErr
}

func validEvaluationFixture(t *testing.T) (EvaluationRequest, *EvaluationAction, *fakeEvaluationRunner, *fakeReservations) {
	t.Helper()
	root := t.TempDir()
	device := gpu.Device{
		UUID: testUUID, Name: "GPU", Architecture: "Ampere", ComputeCapability: "8.0",
		DriverCUDACeiling: "13.0",
		Observations:      []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 0}},
	}
	request := EvaluationRequest{
		Version: EvaluationActionVersion, Backend: nvmsruntime.BackendPodman,
		ManifestSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Executable:     "portage", Argv: []string{"run", "--env", "docker"},
		ExternalEngineToken: ExternalEngineDocker,
		PodmanPipe:          "npipe:////./pipe/podman-machine-default",
		OutputRoot:          root, ReservationPath: filepath.Join(root, "reservations.json"),
		LockInvocation: []string{"portage", "run", "--env", "docker"},
		ResourceManifest: ResourceManifest{
			Version:  gpu.ResourceManifestVersion,
			GPUs:     []gpu.Device{device},
			Artifact: gpu.ArtifactRequirements{CUDAToolkit: "12.8", CompiledKernels: true},
		},
		GPUBindings: []EvaluationGPUBinding{{
			UUID: testUUID, CUDAToolkit: "12.8", CDIDevice: "nvidia.com/gpu=" + string(testUUID),
		}},
		TimeoutMillis: 5_000, MaxOutputBytes: 1024,
	}
	inspector := &fakeEvaluationInspector{inspection: EvaluationInspection{
		Provider: nvmsruntime.BackendPodman, PodmanPipe: request.PodmanPipe,
		Toolkit: "12.8", Devices: []gpu.Device{device},
		CDIDevices: map[gpu.UUID]string{testUUID: "nvidia.com/gpu=" + string(testUUID)},
	}}
	runner := &fakeEvaluationRunner{request: &request, result: gpu.CommandResult{
		ExitCode: 0, Duration: 25 * time.Millisecond, Stdout: []byte("ok"), Stderr: []byte("note"),
	}}
	reservations := &fakeReservations{}
	action := &EvaluationAction{
		Registry: nvmsruntime.NewBackendRegistry(), Inventory: &fakeInventoryProvider{devices: []gpu.Device{device}},
		Inspector: inspector, Runner: runner,
		Reservations: reservations, ReservationTTL: 5*time.Second + EvaluationReservationSkew,
	}
	return request, action, runner, reservations
}

func requireEvaluationCode(t *testing.T, result EvaluationResult, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error", code)
	}
	if result.ErrorCode != code {
		t.Fatalf("error code = %q, want %q (%v)", result.ErrorCode, code, err)
	}
}

func TestEvaluationRequiresExactPodmanWithoutFallback(t *testing.T) {
	t.Run("provider", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		request.Backend = nvmsruntime.BackendNanoVMS
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "provider_rejected")
	})
	t.Run("fallback", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		request.FallbackBackends = []nvmsruntime.BackendID{nvmsruntime.BackendNanoVMS}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "fallback_rejected")
	})
}

func TestEvaluationRejectsMalformedDigestAndBindings(t *testing.T) {
	t.Run("digest", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		request.ManifestSHA256 = "not-a-digest"
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "invalid_manifest_digest")
	})
	t.Run("uuid", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		request.GPUBindings[0].UUID = "GPU-0"
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "gpu_binding_rejected")
	})
	t.Run("toolkit", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		request.GPUBindings[0].CUDAToolkit = ""
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "toolkit_rejected")
	})
}

func TestEvaluationRejectsInspectionMismatch(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	action.Inspector.(*fakeEvaluationInspector).inspection.CDIDevices[testUUID] = "nvidia.com/gpu=all"
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "inspection_mismatch")
}

func TestEvaluationResolvesIdentityOnlyManifestAgainstAuthoritativeInventory(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	request.ResourceManifest.GPUs[0] = gpu.Device{UUID: testUUID}
	request.ResourceManifest.Artifact.CUDAToolkit = "13.0"
	request.GPUBindings[0].CUDAToolkit = "13.0"
	action.Inventory.(*fakeInventoryProvider).devices[0].Name = "NVIDIA GeForce RTX 3090 Ti"
	action.Inventory.(*fakeInventoryProvider).devices[0].Architecture = "Ampere"
	action.Inventory.(*fakeInventoryProvider).devices[0].ComputeCapability = "8.6"
	action.Inspector.(*fakeEvaluationInspector).inspection.Toolkit = "13.0"
	runner.request = &request

	result, err := action.Execute(context.Background(), request)
	if err != nil || !result.Success {
		t.Fatalf("RTX 3090 Ti CUDA 13 evaluation failed: %#v, %v", result, err)
	}
	if result.Provenance.ToolkitVersion != "13.0" ||
		result.Provenance.ToolkitRoot != "" ||
		result.Provenance.ToolkitExecutable != "" {
		t.Fatalf("precompiled toolkit provenance = %#v", result.Provenance)
	}
}

func TestEvaluationInventoryFailuresFailClosedBeforeInspectionOrReservation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(EvaluationRequest, *EvaluationAction)
		code   string
	}{
		{
			name: "unavailable",
			mutate: func(_ EvaluationRequest, action *EvaluationAction) {
				action.Inventory.(*fakeInventoryProvider).err = errors.New("inventory offline")
			},
			code: "inventory_unavailable",
		},
		{
			name: "unknown UUID",
			mutate: func(_ EvaluationRequest, action *EvaluationAction) {
				action.Inventory.(*fakeInventoryProvider).devices = nil
			},
			code: "inventory_mismatch",
		},
		{
			name: "WSL mismatch",
			mutate: func(_ EvaluationRequest, action *EvaluationAction) {
				// The authoritative device is host-visible but absent from the selected distro.
			},
			code: "inventory_mismatch",
		},
		{
			name: "declared capability mismatch",
			mutate: func(request EvaluationRequest, action *EvaluationAction) {
				action.Inventory.(*fakeInventoryProvider).devices[0].ComputeCapability = "8.6"
				_ = request
			},
			code: "inventory_mismatch",
		},
		{
			name: "Pascal CUDA 13",
			mutate: func(_ EvaluationRequest, action *EvaluationAction) {
				device := &action.Inventory.(*fakeInventoryProvider).devices[0]
				device.Name = "NVIDIA GeForce GTX 1080 Ti"
				device.Architecture = "Pascal"
				device.ComputeCapability = "6.1"
			},
			code: "toolkit_rejected",
		},
		{
			name: "driver CUDA ceiling",
			mutate: func(_ EvaluationRequest, action *EvaluationAction) {
				action.Inventory.(*fakeInventoryProvider).devices[0].DriverCUDACeiling = "12.7"
			},
			code: "toolkit_rejected",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, action, runner, reservations := validEvaluationFixture(t)
			if test.name == "WSL mismatch" {
				request.WSLDistribution = "Ubuntu-24.04"
			}
			if test.name == "declared capability mismatch" {
				request.ResourceManifest.GPUs[0].ComputeCapability = "8.0"
			}
			if test.name == "Pascal CUDA 13" {
				request.ResourceManifest.GPUs[0] = gpu.Device{UUID: testUUID}
				request.ResourceManifest.Artifact.CUDAToolkit = "13.0"
				request.GPUBindings[0].CUDAToolkit = "13.0"
			}
			if test.name == "driver CUDA ceiling" {
				request.ResourceManifest.GPUs[0] = gpu.Device{UUID: testUUID}
			}
			test.mutate(request, action)
			result, err := action.Execute(context.Background(), request)
			requireEvaluationCode(t, result, err, test.code)
			if result.Provenance.ToolkitVersion != request.ResourceManifest.Artifact.CUDAToolkit {
				t.Fatalf("precompiled failure lost requested toolkit provenance: %#v", result.Provenance)
			}
			if action.Inspector.(*fakeEvaluationInspector).calls.Load() != 0 || reservations.reserves != 0 {
				t.Fatal("inventory failure reached inspection or reservation")
			}
			if runner.sequence != 0 {
				t.Fatal("inventory failure reached provider execution")
			}
		})
	}
}

func TestEvaluationCapturesTimeoutTruncationAndNonzero(t *testing.T) {
	tests := []struct {
		name   string
		result gpu.CommandResult
		err    error
		code   string
	}{
		{"timeout", gpu.CommandResult{ExitCode: -1, TimedOut: true, Duration: 5 * time.Second}, context.DeadlineExceeded, "action_timeout"},
		{"truncation", gpu.CommandResult{ExitCode: 0, Truncated: true, Stdout: []byte("capped")}, errors.New("bound exceeded"), "output_truncated"},
		{"nonzero", gpu.CommandResult{ExitCode: 17, Stderr: []byte("failure")}, errors.New("exit status 17"), "action_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, action, runner, reservations := validEvaluationFixture(t)
			runner.result, runner.err = test.result, test.err
			result, err := action.Execute(context.Background(), request)
			requireEvaluationCode(t, result, err, test.code)
			if reservations.releases != 1 || !result.Released {
				t.Fatalf("failure did not release reservation: %#v", result)
			}
		})
	}
}

func TestEvaluationSuccessUsesHostPodmanContract(t *testing.T) {
	request, action, runner, reservations := validEvaluationFixture(t)
	request.Environment = map[string]string{"DOCKER_HOST": "bad", "PHENO": "1"}
	action.Inspector.(*fakeEvaluationInspector).inspection.ToolkitRoot = `C:\CUDA\v12.8`
	action.Inspector.(*fakeEvaluationInspector).inspection.ToolkitExecutable = `C:\CUDA\v12.8\bin\nvcc.exe`
	runner.request = &request
	result, err := action.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || !result.Released || reservations.releases != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if runner.command != request.Executable || runner.environment["DOCKER_HOST"] != request.PodmanPipe {
		t.Fatalf("host command/pipe mismatch: %q %#v", runner.command, runner.environment)
	}
	if result.Provenance.EffectiveEngine != "podman" ||
		result.Provenance.ResolvedProvider != nvmsruntime.BackendPodman ||
		result.Provenance.ExecutionPlane != "nanovms" {
		t.Fatalf("untruthful provenance: %#v", result.Provenance)
	}
	if result.Lifecycle.StdoutSHA256 == "" || result.Lifecycle.StderrSHA256 == "" {
		t.Fatal("missing output hashes")
	}
	if result.Provenance.ToolkitRoot != `C:\CUDA\v12.8` ||
		result.Provenance.ToolkitExecutable != `C:\CUDA\v12.8\bin\nvcc.exe` ||
		result.Provenance.ToolkitVersion != "12.8" {
		t.Fatalf("missing resolved toolkit evidence: %#v", result.Provenance)
	}
}

func TestEvaluationRoutesExplicitWSLTransport(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	request.WSLDistribution = "Ubuntu-24.04"
	provider := action.Inventory.(*fakeInventoryProvider)
	provider.devices[0].Observations = append(provider.devices[0].Observations, gpu.Observation{
		Scope: gpu.ScopeWSLDistro, ScopeID: request.WSLDistribution, Index: 0,
	})
	runner.request = &request
	if _, err := action.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if runner.command != "wsl.exe" {
		t.Fatalf("command = %q", runner.command)
	}
	want := []string{"-d", "Ubuntu-24.04", "--", request.Executable}
	for i := range want {
		if runner.arguments[i] != want[i] {
			t.Fatalf("arguments = %#v", runner.arguments)
		}
	}
}

func TestHostInspectionUsesNVCCProofAndRoutesAllWSLProbes(t *testing.T) {
	request, _, _, _ := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = false
	request.WSLDistribution = "Ubuntu-24.04"
	runner := &sequentialCommandRunner{responses: []gpu.CommandResult{
		{Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
		{Stdout: []byte(string(testUUID) + ",0,GPU,8.0,580.1\n")},
		{Stdout: []byte(`<nvidia_smi_log><cuda_version>13.0</cuda_version></nvidia_smi_log>`)},
		{Stdout: []byte("Cuda compilation tools, release 12.8, V12.8.0")},
		{Stdout: []byte("nvidia.com/gpu=" + string(testUUID))},
	}}
	inspection, err := (HostEvaluationInspector{Runner: runner}).Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Toolkit != "12.8" {
		t.Fatalf("toolkit = %q; driver ceiling must not become toolkit proof", inspection.Toolkit)
	}
	if len(runner.commands) != 5 {
		t.Fatalf("commands = %#v", runner.commands)
	}
	for _, command := range runner.commands {
		if len(command) < 5 || command[0] != "wsl.exe" || command[1] != "-d" ||
			command[2] != request.WSLDistribution || command[3] != "--" {
			t.Fatalf("probe bypassed WSL transport: %#v", command)
		}
	}
}

func TestHostInspectionPrecompiledSkipsNVCCAndPreservesProbeOrder(t *testing.T) {
	request, _, _, _ := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CUDAToolkit = "13.0"
	request.ResourceManifest.Artifact.CompiledKernels = true
	runner := &sequentialCommandRunner{responses: []gpu.CommandResult{
		{Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
		{Stdout: []byte(string(testUUID) + ",0,NVIDIA GeForce RTX 3090 Ti,8.6,580.1\n")},
		{Stdout: []byte(`<nvidia_smi_log><cuda_version>13.0</cuda_version></nvidia_smi_log>`)},
		{Stdout: []byte("nvidia.com/gpu=" + string(testUUID))},
	}}

	inspection, err := (HostEvaluationInspector{Runner: runner}).Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Toolkit != "13.0" || inspection.ToolkitRoot != "" || inspection.ToolkitExecutable != "" {
		t.Fatalf("precompiled toolkit evidence = %#v", inspection)
	}
	wantCommands := [][]string{
		{"podman", "info", "--format", "json"},
		{"nvidia-smi", "--query-gpu=uuid,index,name,compute_cap,driver_version", "--format=csv,noheader,nounits"},
		{"nvidia-smi", "-q", "-x"},
		{"nvidia-ctk", "cdi", "list"},
	}
	if !reflect.DeepEqual(runner.commands, wantCommands) {
		t.Fatalf("precompiled inspection commands = %#v, want %#v", runner.commands, wantCommands)
	}
	if len(runner.environments) != 2 {
		t.Fatalf("DOCKER_HOST probe environments = %#v", runner.environments)
	}
	for _, environment := range runner.environments {
		if environment["DOCKER_HOST"] != request.PodmanPipe {
			t.Fatalf("inspection probe DOCKER_HOST = %#v, want %q", environment, request.PodmanPipe)
		}
	}
}

func TestHostInspectionExpandsAllCDIDeviceToInventoryUUIDs(t *testing.T) {
	request, _, _, _ := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = true
	scopedInventory := gpu.ScopedInventory{
		Scope:             gpu.ScopeWindowsHost,
		DriverCUDACeiling: "13.0",
		Devices: []gpu.Device{{
			UUID: testUUID, Name: "NVIDIA GeForce RTX 3090 Ti", Architecture: "Ampere",
			ComputeCapability: "8.6", DriverVersion: "580.1", DriverCUDACeiling: "13.0",
			Observations: []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 0}},
		}},
	}
	runner := &sequentialCommandRunner{responses: []gpu.CommandResult{
		{Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
		{Stdout: []byte("nvidia.com/gpu=all\n")},
	}}
	inspection, err := (HostEvaluationInspector{Runner: runner}).InspectWithInventory(context.Background(), request, &scopedInventory)
	if err != nil {
		t.Fatal(err)
	}
	want := "nvidia.com/gpu=" + string(testUUID)
	if inspection.CDIDevices[testUUID] != want {
		t.Fatalf("CDI devices = %#v, want %q for %s", inspection.CDIDevices, want, testUUID)
	}
}

func TestHostInspectionWithInventorySkipsDuplicateNvidiaSMI(t *testing.T) {
	request, _, _, _ := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = true
	scopedInventory := gpu.ScopedInventory{
		Scope:             gpu.ScopeWindowsHost,
		DriverCUDACeiling: "13.0",
		Devices: []gpu.Device{{
			UUID: testUUID, Name: "NVIDIA GeForce RTX 3090 Ti", Architecture: "Ampere",
			ComputeCapability: "8.6", DriverVersion: "580.1", DriverCUDACeiling: "13.0",
			Observations: []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 0}},
		}},
	}
	runner := &sequentialCommandRunner{responses: []gpu.CommandResult{
		{Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
		{Stdout: []byte("nvidia.com/gpu=" + string(testUUID))},
	}}
	inspection, err := (HostEvaluationInspector{Runner: runner}).InspectWithInventory(context.Background(), request, &scopedInventory)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Devices) != 1 || inspection.Devices[0].UUID != testUUID {
		t.Fatalf("inspection devices = %#v", inspection.Devices)
	}
	wantCommands := [][]string{
		{"podman", "info", "--format", "json"},
		{"nvidia-ctk", "cdi", "list"},
	}
	if !reflect.DeepEqual(runner.commands, wantCommands) {
		t.Fatalf("inventory-backed inspection commands = %#v, want %#v", runner.commands, wantCommands)
	}
}

func TestEvaluationExecuteReusesScopedInventory(t *testing.T) {
	request, _, runner, reservations := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = true
	device := gpu.Device{
		UUID: testUUID, Name: "NVIDIA GeForce RTX 3090 Ti", Architecture: "Ampere",
		ComputeCapability: "8.6", DriverVersion: "580.1", DriverCUDACeiling: "13.0",
		Observations: []gpu.Observation{{Scope: gpu.ScopeWindowsHost, Index: 0}},
	}
	request.ResourceManifest.GPUs = []gpu.Device{device}
	runner = &fakeEvaluationRunner{
		request: &request,
		result:  gpu.CommandResult{ExitCode: 0, Duration: 25 * time.Millisecond, Stdout: []byte("ok")},
	}
	inventoryRunner := &sequentialCommandRunner{responses: []gpu.CommandResult{
		{Stdout: []byte(string(testUUID) + ",0,NVIDIA GeForce RTX 3090 Ti,8.6,580.1\n")},
		{Stdout: []byte(`<nvidia_smi_log><cuda_version>13.0</cuda_version></nvidia_smi_log>`)},
		{Stdout: []byte(`{"host":{"arch":"amd64"}}`)},
		{Stdout: []byte("nvidia.com/gpu=" + string(testUUID))},
	}}
	action := &EvaluationAction{
		Registry: nvmsruntime.NewBackendRegistry(),
		Inventory: gpu.ReconciledInventoryProvider{
			Adapters: []gpu.InventoryAdapter{gpu.WindowsInventoryAdapter{Runner: inventoryRunner}},
		},
		Inspector:      HostEvaluationInspector{Runner: inventoryRunner},
		Runner:         runner,
		Reservations:   reservations,
		ReservationTTL: 5*time.Second + EvaluationReservationSkew,
	}
	result, err := action.Execute(context.Background(), request)
	if err != nil || !result.Success {
		t.Fatalf("evaluation failed: %#v, %v", result, err)
	}
	nvidiaSMI := 0
	for _, command := range inventoryRunner.commands {
		if len(command) > 0 && command[0] == "nvidia-smi" {
			nvidiaSMI++
		}
	}
	if nvidiaSMI != 2 {
		t.Fatalf("nvidia-smi invocations = %d, want 2 (inventory only); commands = %#v", nvidiaSMI, inventoryRunner.commands)
	}
}

func TestEvaluationCompilerRequiredAcceptsResolvedToolkitEvidence(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = false
	root := filepath.Join(t.TempDir(), "cuda", "v12.8")
	inspection := &action.Inspector.(*fakeEvaluationInspector).inspection
	inspection.ToolkitRoot = root
	inspection.ToolkitExecutable = filepath.Join(root, "bin", "nvcc.exe")
	runner.request = &request

	result, err := action.Execute(context.Background(), request)
	if err != nil || !result.Success {
		t.Fatalf("compiler-required evaluation failed: %#v, %v", result, err)
	}
	if result.Provenance.ToolkitRoot != root ||
		result.Provenance.ToolkitExecutable != filepath.Join(root, "bin", "nvcc.exe") ||
		result.Provenance.ToolkitVersion != "12.8" {
		t.Fatalf("compiler toolkit provenance = %#v", result.Provenance)
	}
}

func TestEvaluationCompilerRequiredRejectsMissingToolkitEvidence(t *testing.T) {
	request, action, runner, reservations := validEvaluationFixture(t)
	request.ResourceManifest.Artifact.CompiledKernels = false
	runner.request = &request

	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "toolkit_not_found")
	if runner.sequence != 0 {
		t.Fatal("missing compiler evidence reached provider execution")
	}
	if reservations.reserves != 1 || reservations.releases != 1 || !result.Released {
		t.Fatalf("inspect-time toolkit failure reservation state = reserves:%d releases:%d released:%v",
			reservations.reserves, reservations.releases, result.Released)
	}
}

func TestToolkitDiscoveryPrecedenceAndEvidence(t *testing.T) {
	base := t.TempDir()
	configuredRoot := filepath.Join(base, "configured", "cuda")
	versionEnvRoot := filepath.Join(base, "version-env", "cuda")
	cudaPathRoot := filepath.Join(base, "cuda-path")
	configuredNVCC := filepath.Join(configuredRoot, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		configuredNVCC: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.1")}},
	}}
	inspector := HostEvaluationInspector{
		Runner: runner, ToolkitRoot: configuredRoot,
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

	resolved, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Root != configuredRoot || resolved.Executable != configuredNVCC || resolved.Version != "13.0" {
		t.Fatalf("resolved toolkit = %#v", resolved)
	}
	if len(runner.commands) != 1 || runner.commands[0][0] != configuredNVCC {
		t.Fatalf("discovery did not stop at configured root: %#v", runner.commands)
	}
}

func TestToolkitDiscoveryVersionEnvironmentAndStandardWindowsPath(t *testing.T) {
	nativeRoot := filepath.Join(t.TempDir(), "cuda", "v13.0")
	tests := []struct {
		name         string
		lookup       func(string) (string, bool)
		programRoots []string
		wantRoot     string
		windowsOnly  bool
	}{
		{
			name: "version environment",
			lookup: func(key string) (string, bool) {
				if key == "CUDA_PATH_V13_0" {
					return nativeRoot, true
				}
				return "", false
			},
			wantRoot: nativeRoot,
		},
		{
			name:         "standard path",
			lookup:       func(string) (string, bool) { return "", false },
			programRoots: []string{`C:\Program Files`},
			wantRoot:     `C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v13.0`,
			windowsOnly:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("Windows standard-path contract")
			}
			nvcc := filepath.Join(test.wantRoot, "bin", "nvcc.exe")
			runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
				nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
			}}
			inspector := HostEvaluationInspector{
				Runner: runner, LookupEnv: test.lookup, ProgramFilesRoots: test.programRoots,
			}
			resolved, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
				ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if resolved.Root != test.wantRoot || resolved.Executable != nvcc {
				t.Fatalf("resolved toolkit = %#v", resolved)
			}
		})
	}
}

func TestToolkitDiscoveryUsesCUDAPathOnlyWhenVersionMatches(t *testing.T) {
	base := t.TempDir()
	cudaPathRoot := filepath.Join(base, "cuda-path")
	programFilesRoot := filepath.Join(base, "program-files")
	standardRoot := filepath.Join(programFilesRoot, "NVIDIA GPU Computing Toolkit", "CUDA", "v13.0")
	cudaPathNVCC := filepath.Join(cudaPathRoot, "bin", "nvcc.exe")
	standardNVCC := filepath.Join(standardRoot, "bin", "nvcc.exe")
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		cudaPathNVCC: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 12.8, V12.8.0")}},
		standardNVCC: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	inspector := HostEvaluationInspector{
		Runner: runner,
		LookupEnv: func(key string) (string, bool) {
			if key == "CUDA_PATH" {
				return cudaPathRoot, true
			}
			return "", false
		},
		ProgramFilesRoots: []string{programFilesRoot},
	}
	resolved, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Root != standardRoot || resolved.Version != "13.0" {
		t.Fatalf("resolved mismatched CUDA_PATH: %#v", resolved)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("toolkit probes = %#v", runner.commands)
	}
}

func TestToolkitDiscoveryFailsClosed(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "cuda", "v13.0")
	nvcc := filepath.Join(root, "bin", "nvcc.exe")
	tests := []struct {
		name      string
		responses map[string]toolkitCommandResponse
		roots     []string
		code      string
	}{
		{
			name: "version mismatch",
			responses: map[string]toolkitCommandResponse{
				nvcc: {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 12.8, V12.8.0")}},
			},
			code: "toolkit_version_mismatch",
		},
		{name: "missing nvcc", responses: map[string]toolkitCommandResponse{}, code: "toolkit_not_found"},
		{
			name: "timeout",
			responses: map[string]toolkitCommandResponse{
				nvcc: {result: gpu.CommandResult{TimedOut: true}, err: context.DeadlineExceeded},
			},
			code: "toolkit_inspection_timeout",
		},
		{
			name: "ambiguous installations",
			roots: []string{
				filepath.Join(base, "program-files-a"),
				filepath.Join(base, "program-files-b"),
			},
			responses: map[string]toolkitCommandResponse{
				filepath.Join(base, "program-files-a", "NVIDIA GPU Computing Toolkit", "CUDA", "v13.0", "bin", "nvcc.exe"): {
					result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")},
				},
				filepath.Join(base, "program-files-b", "NVIDIA GPU Computing Toolkit", "CUDA", "v13.0", "bin", "nvcc.exe"): {
					result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")},
				},
			},
			code: "toolkit_ambiguous",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inspector := HostEvaluationInspector{
				Runner: &toolkitCommandRunner{responses: test.responses},
				LookupEnv: func(key string) (string, bool) {
					if len(test.roots) == 0 && key == "CUDA_PATH_V13_0" {
						return root, true
					}
					return "", false
				},
				ProgramFilesRoots: test.roots,
			}
			_, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
				ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
			})
			var evaluationErr *EvaluationError
			if !errors.As(err, &evaluationErr) || evaluationErr.Code != test.code {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestToolkitDiscoveryUsesSelectedWSLProvider(t *testing.T) {
	provider := &fakeWSLToolkitProvider{roots: []string{"/opt/cuda-13.0"}}
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		"wsl.exe": {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	inspector := HostEvaluationInspector{
		Runner: runner, LookupEnv: func(string) (string, bool) { return "", false },
		WSLToolkitProvider: provider,
	}
	resolved, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		WSLDistribution:  "Ubuntu-24.04",
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.distribution != "Ubuntu-24.04" || provider.version != "13.0" {
		t.Fatalf("provider selection = %q %q", provider.distribution, provider.version)
	}
	if resolved.Root != "/opt/cuda-13.0" || resolved.Executable != "/opt/cuda-13.0/bin/nvcc" {
		t.Fatalf("resolved toolkit = %#v", resolved)
	}
	want := []string{"wsl.exe", "-d", "Ubuntu-24.04", "--", "/opt/cuda-13.0/bin/nvcc", "--version"}
	if len(runner.commands) != 1 || !reflect.DeepEqual(runner.commands[0], want) {
		t.Fatalf("WSL toolkit probe = %#v, want %#v", runner.commands, want)
	}
}

func TestToolkitDiscoveryWSLIgnoresHostCUDAPaths(t *testing.T) {
	provider := &fakeWSLToolkitProvider{roots: []string{"/opt/cuda-13.0"}}
	runner := &toolkitCommandRunner{responses: map[string]toolkitCommandResponse{
		"wsl.exe": {result: gpu.CommandResult{Stdout: []byte("Cuda compilation tools, release 13.0, V13.0.0")}},
	}}
	inspector := HostEvaluationInspector{
		Runner: runner,
		LookupEnv: func(key string) (string, bool) {
			if key == "CUDA_PATH_V13_0" {
				return `C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v13.0`, true
			}
			return "", false
		},
		WSLToolkitProvider: provider,
	}
	resolved, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		WSLDistribution:  "Ubuntu-24.04",
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	if err != nil || resolved.Root != "/opt/cuda-13.0" {
		t.Fatalf("selected WSL provider was blocked by host environment: %#v, %v", resolved, err)
	}
}

func TestToolkitDiscoveryWSLProviderTimeout(t *testing.T) {
	inspector := HostEvaluationInspector{
		Runner: &toolkitCommandRunner{}, LookupEnv: func(string) (string, bool) { return "", false },
		WSLToolkitProvider: &fakeWSLToolkitProvider{err: context.DeadlineExceeded},
	}
	_, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		WSLDistribution:  "Ubuntu-24.04",
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) || evaluationErr.Code != "toolkit_inspection_timeout" {
		t.Fatalf("error = %v, want toolkit_inspection_timeout", err)
	}
}

func TestToolkitDiscoveryRejectsAmbiguousEnvironmentKeys(t *testing.T) {
	inspector := HostEvaluationInspector{
		Runner: &toolkitCommandRunner{}, LookupEnv: func(string) (string, bool) { return "", false },
	}
	_, err := inspector.resolveToolkit(context.Background(), EvaluationRequest{
		Environment: map[string]string{
			"CUDA_PATH_V13_0": filepath.Join(t.TempDir(), "cuda-a"),
			"cuda_path_v13_0": filepath.Join(t.TempDir(), "cuda-b"),
		},
		ResourceManifest: ResourceManifest{Artifact: gpu.ArtifactRequirements{CUDAToolkit: "13.0"}},
	})
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) || evaluationErr.Code != "toolkit_ambiguous" {
		t.Fatalf("error = %v, want toolkit_ambiguous", err)
	}
}

func TestEvaluationEarlyFailureReleasesReservationAfterInspect(t *testing.T) {
	request, action, _, reservations := validEvaluationFixture(t)
	action.Inspector.(*fakeEvaluationInspector).err = evaluationError(CodeToolkitNotFound, "CUDA toolkit unavailable")
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "toolkit_not_found")
	if !result.Released || reservations.reserves != 1 || reservations.releases != 1 {
		t.Fatalf("inspect failure cleanup state = released:%v reserves:%d releases:%d", result.Released, reservations.reserves, reservations.releases)
	}
}

func TestEvaluationRejectsAmbiguousJobOutput(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	runner.request = nil
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "ambiguous_job_output")
}

func TestEvaluationPrefersActionFailedOverAmbiguousJobOutput(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	runner.request = nil
	runner.result.ExitCode = 7
	runner.err = errors.New("portage failed")
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "action_failed")
}

func TestEvaluationRejectsReservationStorePathMismatch(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	action.Reservations = &gpu.ReservationStore{Path: filepath.Join(t.TempDir(), "other-reservations.json")}
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "invalid_reservation_path")
	if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
		t.Fatal("reservation path mismatch reached inventory")
	}
}

func TestLockInvocationsMatchPortagePathAndRedactedSecrets(t *testing.T) {
	expected := []string{
		"portage", "run", "--agent-env", "OPENAI_API_KEY=local-dummy-key",
	}
	actual := []string{
		`C:\Users\koosh\portage\.venv\Scripts\portage`, "run", "--agent-env", "OPENAI_API_KEY=loca****key",
	}
	if !lockInvocationsMatch(expected, actual) {
		t.Fatal("expected Harbor lock invocation to match delegated request")
	}
	if lockInvocationsMatch(expected, []string{"portage", "different-job"}) {
		t.Fatal("expected materially different invocation to mismatch")
	}
}

func TestEvaluationRejectsJobLockInvocationMismatch(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	written := request
	written.LockInvocation = []string{"portage", "different-job"}
	runner.request = &written
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "job_lock_mismatch")
}

func TestEvaluationCleanupFailurePreservesEvidence(t *testing.T) {
	request, action, _, reservations := validEvaluationFixture(t)
	reservations.releaseErr = errors.New("release blocked")
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "cleanup_failed")
	if result.Lifecycle.Stdout != "ok" || result.Provenance.JobDirectory == "" || result.Released {
		t.Fatalf("failure evidence lost: %#v", result)
	}
}

func TestEvaluationCleanupFailurePreservesPrimaryActionCode(t *testing.T) {
	request, action, runner, reservations := validEvaluationFixture(t)
	runner.result.ExitCode = 7
	runner.err = errors.New("portage failed")
	reservations.releaseErr = errors.New("release blocked")
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "action_failed")
	if result.Released || !strings.Contains(result.ErrorMessage, "cleanup_failed") {
		t.Fatalf("primary failure lost cleanup evidence: %#v", result)
	}
}

func TestEvaluationRejectsUnsafeReservationPaths(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	rootPath := string(filepath.Separator)
	if volume := filepath.VolumeName(request.ReservationPath); volume != "" {
		rootPath = volume + string(filepath.Separator)
	}
	paths := []string{
		"",
		rootPath,
		filepath.Join(request.OutputRoot, "child") + string(filepath.Separator),
		request.OutputRoot + string(filepath.Separator) + ".." + string(filepath.Separator) + "reservations.json",
	}
	if runtime.GOOS == "windows" {
		paths = append(paths, `\\server\share\reservations.json`)
	}
	for _, path := range paths {
		t.Run(fmt.Sprintf("%q", path), func(t *testing.T) {
			candidate := request
			candidate.ReservationPath = path
			result, err := action.Execute(context.Background(), candidate)
			requireEvaluationCode(t, result, err, "invalid_reservation_path")
		})
	}
}

func TestEvaluationRequiresReservationSkewBeyondTimeout(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	action.ReservationTTL = time.Duration(request.TimeoutMillis) * time.Millisecond
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "invalid_limits")
}

func TestEvaluationSerializesSharedOutputRoot(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	runner.delay = 40 * time.Millisecond
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := action.Execute(context.Background(), request)
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	if runner.maxActive != 1 {
		t.Fatalf("maximum concurrent runners = %d, want 1", runner.maxActive)
	}
}

func TestEvaluationCreatesAbsentNestedOutputRootAndRecordsSpace(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	root := filepath.Join(t.TempDir(), "nested", "harbor")
	request.OutputRoot = root
	runner.request = &request
	action.Filesystem = &injectedEvaluationFilesystem{root: root, available: 987654321}

	result, err := action.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Provenance.OutputRootCreated {
		t.Fatal("new output root was not recorded as NanoVMS-created")
	}
	if result.Provenance.OutputRootAvailableBytes == nil || *result.Provenance.OutputRootAvailableBytes != 987654321 {
		t.Fatalf("available-space evidence = %#v", result.Provenance.OutputRootAvailableBytes)
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		t.Fatalf("nested output root was not created: %v", statErr)
	}
}

func TestEvaluationConcurrentOutputRootCreationHasSingleCreator(t *testing.T) {
	request, action, runner, _ := validEvaluationFixture(t)
	root := filepath.Join(t.TempDir(), "nested", "harbor")
	request.OutputRoot = root
	runner.request = &request
	action.Filesystem = &injectedEvaluationFilesystem{root: root}
	runner.delay = 20 * time.Millisecond

	results := make(chan EvaluationResult, 2)
	errorsSeen := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := action.Execute(context.Background(), request)
			results <- result
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
	}
	creators := 0
	for result := range results {
		if result.Provenance.OutputRootCreated {
			creators++
		}
	}
	if creators != 1 {
		t.Fatalf("NanoVMS-created results = %d, want 1", creators)
	}
	if runner.maxActive != 1 {
		t.Fatalf("maximum concurrent runners = %d, want 1", runner.maxActive)
	}
}

func TestEvaluationCoordinatesCleanupWithConcurrentRootCreation(t *testing.T) {
	requestA, actionA, _, _ := validEvaluationFixture(t)
	requestB, actionB, runnerB, _ := validEvaluationFixture(t)
	root := filepath.Join(t.TempDir(), "nested", "harbor")
	requestA.OutputRoot = root
	requestB.OutputRoot = root
	runnerB.request = &requestB

	aAvailable := make(chan struct{})
	bAvailable := make(chan struct{})
	aDone := make(chan struct{})
	actionA.Filesystem = &injectedEvaluationFilesystem{
		root:         root,
		availableErr: errors.New("space unavailable"),
		onAvailable: func(string) {
			close(aAvailable)
			select {
			case <-bAvailable:
			case <-time.After(250 * time.Millisecond):
			}
		},
	}
	actionB.Filesystem = &injectedEvaluationFilesystem{
		root: root,
		onAvailable: func(string) {
			close(bAvailable)
			<-aDone
		},
	}

	errorA := make(chan error, 1)
	go func() {
		result, err := actionA.Execute(context.Background(), requestA)
		if err != nil && result.ErrorCode != "output_root_space_failed" {
			errorA <- fmt.Errorf("creator failure code = %q: %w", result.ErrorCode, err)
		} else {
			errorA <- err
		}
		close(aDone)
	}()
	<-aAvailable

	resultB, errB := actionB.Execute(context.Background(), requestB)
	errA := <-errorA
	var evaluationErr *EvaluationError
	if !errors.As(errA, &evaluationErr) || evaluationErr.Code != "output_root_space_failed" {
		t.Fatalf("creator error = %v, want output_root_space_failed", errA)
	}
	if errB != nil {
		t.Fatalf("concurrent evaluation failed after coordinated cleanup: %v (%#v)", errB, resultB)
	}
}

func TestEvaluationPreservesPreexistingOutputRoot(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	marker := filepath.Join(request.OutputRoot, "user-content")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := action.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Provenance.OutputRootCreated {
		t.Fatal("pre-existing output root was recorded as created")
	}
	if data, readErr := os.ReadFile(marker); readErr != nil || string(data) != "keep" {
		t.Fatalf("pre-existing content changed: %q, %v", data, readErr)
	}
}

func TestEvaluationOutputRootFailuresPrecedeInventory(t *testing.T) {
	t.Run("file collision", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "collision")
		if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		request.OutputRoot = root
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_collision")
		if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
			t.Fatal("file collision reached inventory provider")
		}
	})

	t.Run("create permission", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "denied")
		request.OutputRoot = root
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root, mkdirErr: os.ErrPermission,
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_create_failed")
		if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
			t.Fatal("create failure reached inventory provider")
		}
	})

	t.Run("lock permission", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "new-root")
		request.OutputRoot = root
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root, openErr: os.ErrPermission,
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_lock_failed")
		if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
			t.Fatal("lock failure reached inventory provider")
		}
		if _, statErr := os.Stat(root); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("created root remains after lock failure: %v", statErr)
		}
	})
}

func TestEvaluationRejectsUnsafeOutputRootPathsBeforeInventory(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	rootPath := string(filepath.Separator)
	if volume := filepath.VolumeName(request.OutputRoot); volume != "" {
		rootPath = volume + string(filepath.Separator)
	}
	paths := []string{
		"",
		rootPath,
		filepath.Join(request.OutputRoot, "child") + string(filepath.Separator),
		request.OutputRoot + string(filepath.Separator) + ".." + string(filepath.Separator) + "other",
	}
	if runtime.GOOS == "windows" {
		paths = append(paths, `\\server\share`)
	}
	for _, path := range paths {
		t.Run(fmt.Sprintf("%q", path), func(t *testing.T) {
			candidate := request
			candidate.OutputRoot = path
			result, err := action.Execute(context.Background(), candidate)
			requireEvaluationCode(t, result, err, "invalid_output_root")
		})
	}
	if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
		t.Fatal("invalid output root reached inventory provider")
	}
}

func TestEvaluationRejectsSymlinkOutputRootEscapeWhereSupported(t *testing.T) {
	request, action, _, _ := validEvaluationFixture(t)
	base := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(base, "escape")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	request.OutputRoot = filepath.Join(link, "harbor")
	result, err := action.Execute(context.Background(), request)
	requireEvaluationCode(t, result, err, "invalid_output_root")
	if action.Inventory.(*fakeInventoryProvider).calls.Load() != 0 {
		t.Fatal("symlink escape reached inventory provider")
	}
}

func TestEvaluationPreActionFailureCleansOnlyEmptyCreatedRoot(t *testing.T) {
	t.Run("empty root", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "harbor")
		request.OutputRoot = root
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root, availableErr: errors.New("space unavailable"),
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_space_failed")
		if _, statErr := os.Stat(root); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("empty NanoVMS-created root remains: %v", statErr)
		}
	})

	t.Run("owned lock", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "harbor")
		request.OutputRoot = root
		action.Inventory.(*fakeInventoryProvider).err = errors.New("inventory unavailable")
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "inventory_unavailable")
		if _, statErr := os.Stat(root); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("coordinated empty root remains after pre-action failure: %v", statErr)
		}
	})

	t.Run("user content", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "harbor")
		request.OutputRoot = root
		content := filepath.Join(root, "user-content")
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root,
			onAvailable: func(string) {
				if err := os.WriteFile(content, []byte("preserve"), 0o600); err != nil {
					t.Errorf("write user content: %v", err)
				}
			},
			availableErr: errors.New("space unavailable"),
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_cleanup_failed")
		if data, readErr := os.ReadFile(content); readErr != nil || string(data) != "preserve" {
			t.Fatalf("user content was not preserved: %q, %v", data, readErr)
		}
	})

	t.Run("cleanup permission", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "harbor")
		request.OutputRoot = root
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root, removeRootErr: os.ErrPermission, availableErr: errors.New("space unavailable"),
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_cleanup_failed")
		if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
			t.Fatalf("cleanup-failure evidence was not preserved: %v", statErr)
		}
	})

	t.Run("unowned concurrent lock", func(t *testing.T) {
		request, action, _, _ := validEvaluationFixture(t)
		root := filepath.Join(t.TempDir(), "harbor")
		request.OutputRoot = root
		lockPath := filepath.Join(root, outputRootLockFilename)
		action.Filesystem = &injectedEvaluationFilesystem{
			root: root,
			onAvailable: func(string) {
				file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
				if err != nil {
					t.Errorf("create concurrent lock: %v", err)
					return
				}
				_ = file.Close()
			},
			availableErr: errors.New("space unavailable"),
		}
		result, err := action.Execute(context.Background(), request)
		requireEvaluationCode(t, result, err, "output_root_cleanup_failed")
		if _, statErr := os.Stat(lockPath); statErr != nil {
			t.Fatalf("unowned lock evidence was removed: %v", statErr)
		}
	})
}
