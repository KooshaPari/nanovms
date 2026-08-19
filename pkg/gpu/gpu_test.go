// SPDX-License-Identifier: MIT OR Apache-2.0
package gpu

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testUUIDA UUID = "GPU-11111111-1111-1111-1111-111111111111"
	testUUIDB UUID = "GPU-22222222-2222-2222-2222-222222222222"
)

type fakeCommandRunner struct {
	mu        sync.Mutex
	responses []CommandResult
	errors    []error
	commands  [][]string
}

func (runner *fakeCommandRunner) Run(_ context.Context, name string, args ...string) (CommandResult, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.commands = append(runner.commands, append([]string{name}, args...))
	if len(runner.responses) == 0 {
		return CommandResult{}, errors.New("unexpected command")
	}
	result := runner.responses[0]
	runner.responses = runner.responses[1:]
	var err error
	if len(runner.errors) > 0 {
		err = runner.errors[0]
		runner.errors = runner.errors[1:]
	}
	return result, err
}

func inventoryResponses(csv string) []CommandResult {
	return []CommandResult{
		{Stdout: []byte(csv)},
		{Stdout: []byte(`<nvidia_smi_log><cuda_version>13.0</cuda_version></nvidia_smi_log>`)},
	}
}

func TestInventoryReconcilesSwappedIndicesByUUID(t *testing.T) {
	hostRunner := &fakeCommandRunner{responses: inventoryResponses(
		string(testUUIDA) + ", 0, NVIDIA GeForce GTX 1080 Ti, 6.1, 580.1\n" +
			string(testUUIDB) + ", 1, NVIDIA GeForce RTX 3090 Ti, 8.6, 580.1\n")}
	wslRunner := &fakeCommandRunner{responses: inventoryResponses(
		string(testUUIDB) + ", 0, NVIDIA GeForce RTX 3090 Ti, 8.6, 580.1\n" +
			string(testUUIDA) + ", 1, NVIDIA GeForce GTX 1080 Ti, 6.1, 580.1\n")}

	host, err := (WindowsInventoryAdapter{Runner: hostRunner}).Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wsl, err := (WSLInventoryAdapter{Runner: wslRunner, Distribution: "Ubuntu"}).Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	devices, err := Reconcile(host, wsl)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 || devices[0].UUID != testUUIDA {
		t.Fatalf("unexpected reconciled devices: %#v", devices)
	}
	if devices[1].Architecture != "Ampere" || devices[1].ComputeCapability != "8.6" {
		t.Fatalf("RTX 3090 Ti facts = %#v", devices[1])
	}
	if got := []int{devices[0].Observations[0].Index, devices[0].Observations[1].Index}; !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatalf("UUID A indices = %v, want [0 1]", got)
	}
	if !reflect.DeepEqual(wslRunner.commands[0][:6], []string{"wsl.exe", "-d", "Ubuntu", "--", "nvidia-smi", "--query-gpu=uuid,index,name,compute_cap,driver_version"}) {
		t.Fatalf("unexpected WSL command: %v", wslRunner.commands[0])
	}
}

func TestInventoryAllowsSubsetVisibility(t *testing.T) {
	host, err := parseInventoryCSV([]byte(
		string(testUUIDA)+",0,GTX 1080 Ti,6.1,580\n"+
			string(testUUIDB)+",1,RTX 3090 Ti,8.6,580\n"), ScopeWindowsHost, "", "13.0")
	if err != nil {
		t.Fatal(err)
	}
	wsl, err := parseInventoryCSV([]byte(string(testUUIDB)+",0,RTX 3090 Ti,8.6,580\n"), ScopeWSLDistro, "Ubuntu", "13.0")
	if err != nil {
		t.Fatal(err)
	}
	devices, err := Reconcile(host, wsl)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 || len(devices[0].Observations) != 1 || len(devices[1].Observations) != 2 {
		t.Fatalf("subset visibility was not preserved: %#v", devices)
	}
}

func TestInventoryRejectsMalformedMissingAndDuplicateUUID(t *testing.T) {
	tests := map[string]string{
		"malformed": "GPU-not-a-uuid,0,GPU,8.6,580\n",
		"missing":   ",0,GPU,8.6,580\n",
		"duplicate": string(testUUIDA) + ",0,GPU,8.6,580\n" + string(testUUIDA) + ",1,GPU,8.6,580\n",
	}
	for name, csv := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseInventoryCSV([]byte(csv), ScopeWindowsHost, "", "13.0"); err == nil {
				t.Fatal("expected invalid UUID inventory to fail")
			}
		})
	}
}

func TestCUDACompatibilityPascalAndAmpere(t *testing.T) {
	artifact13 := ArtifactRequirements{CUDAToolkit: "13.0", CompiledKernels: true}
	pascal := Device{UUID: testUUIDA, Architecture: "Pascal", ComputeCapability: "6.1", DriverCUDACeiling: "13.0"}
	ampere := Device{UUID: testUUIDB, Architecture: "Ampere", ComputeCapability: "8.6", DriverCUDACeiling: "13.0"}
	if err := ValidateCompatibility(pascal, artifact13); err == nil {
		t.Fatal("expected Pascal CUDA 13 compiled-kernel plan rejection")
	}
	if err := ValidateCompatibility(ampere, artifact13); err != nil {
		t.Fatalf("Ampere CUDA 13 should be accepted: %v", err)
	}
	if err := ValidateCompatibility(pascal, ArtifactRequirements{CUDAToolkit: "12.8", CompiledKernels: true}); err != nil {
		t.Fatalf("declared CUDA 12.x Pascal artifact should be accepted: %v", err)
	}
	if err := ValidateCompatibility(pascal, ArtifactRequirements{CUDAToolkit: "13.0", CompiledKernels: false}); err == nil {
		t.Fatal("expected Pascal CUDA 13 compiler-required plan rejection")
	}
}

func TestDriverDisplayIsOnlyACeiling(t *testing.T) {
	device := Device{UUID: testUUIDB, Architecture: "Ampere", DriverCUDACeiling: "12.4"}
	if err := ValidateCompatibility(device, ArtifactRequirements{CUDAToolkit: "12.5", CompiledKernels: true}); err == nil || !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("expected driver ceiling rejection, got %v", err)
	}
	if err := ValidateCompatibility(device, ArtifactRequirements{CUDAToolkit: "12.5", CompiledKernels: false}); err == nil || !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("expected driver ceiling rejection for compiler-required artifact, got %v", err)
	}
}

func TestExecRunnerTimeout(t *testing.T) {
	if slices.Contains(os.Args, "gpu-timeout-helper") {
		time.Sleep(2 * time.Second)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner := ExecRunner{Timeout: 25 * time.Millisecond, MaxOutput: 1024}
	_, err = runner.Run(context.Background(), executable, "-test.run=TestExecRunnerTimeout", "--", "gpu-timeout-helper")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded timeout, got %v", err)
	}
}

func TestConcurrentReservationHasOneWinner(t *testing.T) {
	storePath := t.TempDir() + string(os.PathSeparator) + "reservations.json"
	stores := []*ReservationStore{{Path: storePath}, {Path: storePath}}
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := range stores {
		go func(index int) {
			<-start
			_, err := stores[index].Reserve(context.Background(), []UUID{testUUIDA}, "owner", time.Minute)
			results <- err
		}(i)
	}
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("expected one reservation winner, got %v and %v", first, second)
	}
}

func TestReservationEnforcesOwnershipAndExpiry(t *testing.T) {
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	store := &ReservationStore{
		Path: t.TempDir() + string(os.PathSeparator) + "reservations.json",
		Now:  func() time.Time { return now },
	}
	lease, err := store.Reserve(context.Background(), []UUID{testUUIDA}, "owner-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	wrongLease := lease
	wrongLease.Token = strings.Repeat("0", len(lease.Token))
	if err := store.Release(context.Background(), wrongLease); err == nil {
		t.Fatal("expected ownership-token mismatch")
	}
	now = now.Add(2 * time.Minute)
	if _, err := store.Reserve(context.Background(), []UUID{testUUIDA}, "owner-b", time.Minute); err != nil {
		t.Fatalf("expired reservation was not reusable: %v", err)
	}
}

func TestResourceManifestIsDeterministic(t *testing.T) {
	deviceA := Device{UUID: testUUIDA, Name: "A", Observations: []Observation{{Scope: ScopeWindowsHost, Index: 1}}}
	deviceB := Device{UUID: testUUIDB, Name: "B", Observations: []Observation{{Scope: ScopeWindowsHost, Index: 0}}}
	left, err := (ResourceManifest{Version: ResourceManifestVersion, GPUs: []Device{deviceB, deviceA}}).CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	right, err := (ResourceManifest{Version: ResourceManifestVersion, GPUs: []Device{deviceA, deviceB}}).CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatalf("manifest serialization is not deterministic:\n%s\n%s", left, right)
	}
}

func TestResourceManifestAllowsInventoryOwnedCompatibilityFacts(t *testing.T) {
	manifest := ResourceManifest{
		Version:  ResourceManifestVersion,
		GPUs:     []Device{{UUID: testUUIDB}},
		Artifact: ArtifactRequirements{CUDAToolkit: "13.0", CompiledKernels: true},
	}
	if _, err := manifest.CanonicalJSON(); err != nil {
		t.Fatalf("identity-only manifest should defer hardware compatibility to inventory: %v", err)
	}
}
