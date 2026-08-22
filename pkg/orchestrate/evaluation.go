// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

// ResourceManifest is a type alias for gpu.ResourceManifest, allowing the
// orchestrate package to reference it without the gpu. prefix.
type ResourceManifest = gpu.ResourceManifest

const (
	EvaluationActionVersion = "nanovms.io/evaluation-action/v1"
	ExternalEngineDocker    = "docker"
	EffectiveEnginePodman   = "podman"
	ExecutionPlaneNanoVMS   = "nanovms"
	JobLockFilename         = "lock.json"

	MaxEvaluationTimeout     = 24 * time.Hour
	MaxEvaluationOutput      = 16 << 20
	EvaluationCleanupTimeout = 5 * time.Second
	// EvaluationReservationSkew covers inventory/inspect/reserve overhead so the
	// GPU lease cannot expire while the bounded command is still running.
	EvaluationReservationSkew = 60 * time.Second
)

// EvaluationRequest is the complete, fail-closed host action boundary.
// Executable is launched directly on the host (or through an explicit WSL
// transport); NanoVMS does not create an OCI container for this action.
type EvaluationRequest struct {
	Version             string                  `json:"version"`
	Backend             nvmsruntime.BackendID   `json:"backend"`
	FallbackBackends    []nvmsruntime.BackendID `json:"fallback_backends,omitempty"`
	ManifestSHA256      string                  `json:"manifest_sha256"`
	Executable          string                  `json:"executable"`
	Argv                []string                `json:"argv"`
	Environment         map[string]string       `json:"environment,omitempty"`
	ExternalEngineToken string                  `json:"external_engine_token"`
	PodmanPipe          string                  `json:"podman_pipe"`
	WSLDistribution     string                  `json:"wsl_distribution,omitempty"`
	OutputRoot          string                  `json:"output_root"`
	ReservationPath     string                  `json:"reservation_path"`
	LockInvocation      []string                `json:"lock_invocation"`
	ResourceManifest    ResourceManifest        `json:"resource_manifest"`
	GPUBindings         []EvaluationGPUBinding  `json:"gpu_bindings"`
	TimeoutMillis       int64                   `json:"timeout_millis"`
	MaxOutputBytes      int                     `json:"max_output_bytes"`
}

// EvaluationGPUBinding ties one canonical UUID to the requested artifact
// toolkit and a concrete CDI device name. Driver-reported CUDA ceilings are
// compatibility bounds, not installed compiler evidence.
type EvaluationGPUBinding struct {
	UUID        gpu.UUID `json:"uuid"`
	CUDAToolkit string   `json:"cuda_toolkit"`
	CDIDevice   string   `json:"cdi_device"`
}

// EvaluationInspection is independently observed host state.
type EvaluationInspection struct {
	Provider   nvmsruntime.BackendID `json:"provider"`
	PodmanPipe string                `json:"podman_pipe"`
	Toolkit    string                `json:"cuda_toolkit"`
	// ToolkitRoot and ToolkitExecutable are empty when precompiled kernels make
	// a host compiler unnecessary; empty means not required, not unavailable.
	ToolkitRoot       string              `json:"toolkit_root,omitempty"`
	ToolkitExecutable string              `json:"toolkit_executable,omitempty"`
	Devices           []gpu.Device        `json:"devices"`
	CDIDevices        map[gpu.UUID]string `json:"cdi_devices"`
}

// EvaluationLifecycle records bounded process evidence on both success and
// failure. Hashes cover exactly the capped byte strings returned here.
type EvaluationLifecycle struct {
	ExitCode     int    `json:"exit_code"`
	DurationMS   int64  `json:"duration_ms"`
	TimedOut     bool   `json:"timed_out"`
	Truncated    bool   `json:"truncated"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	StdoutSHA256 string `json:"stdout_sha256"`
	StderrSHA256 string `json:"stderr_sha256"`
}

// EvaluationProvenance records the actual execution route rather than the
// historical external schema token.
type EvaluationProvenance struct {
	ManifestSHA256           string                `json:"manifest_sha256"`
	EffectiveEngine          string                `json:"effective_engine"`
	ResolvedProvider         nvmsruntime.BackendID `json:"resolved_provider"`
	ExecutionPlane           string                `json:"execution_plane"`
	PodmanPipe               string                `json:"podman_pipe"`
	GPUUUIDs                 []gpu.UUID            `json:"gpu_uuids"`
	JobDirectory             string                `json:"job_directory,omitempty"`
	CandidateJobDirectories  []string              `json:"-"`
	OutputRoot               string                `json:"-"`
	OutputLockPath           string                `json:"-"`
	CoordinatorLockPath      string                `json:"-"`
	ReservationPath          string                `json:"-"`
	ReservationToken         string                `json:"-"`
	ReservationExpiresAt     string                `json:"-"`
	LockInvocation           []string              `json:"-"`
	CommandSHA256            string                `json:"-"`
	OutputRootCreated        bool                  `json:"output_root_created"`
	OutputRootAvailableBytes *uint64               `json:"output_root_available_bytes,omitempty"`
	// OutputRootCleanup records whether a NanoVMS-created root was removed or
	// preserved after a pre-action failure. Empty means cleanup was not attempted.
	OutputRootCleanup string `json:"-"`
	// Empty toolkit paths mean the precompiled artifact did not require a host
	// compiler; ToolkitVersion still records the requested compatibility target.
	ToolkitRoot       string `json:"toolkit_root,omitempty"`
	ToolkitExecutable string `json:"toolkit_executable,omitempty"`
	ToolkitVersion    string `json:"toolkit_version,omitempty"`
}

// EvaluationResult is always safe to serialize as failure evidence.
type EvaluationResult struct {
	Version      string               `json:"version"`
	Success      bool                 `json:"success"`
	ErrorCode    string               `json:"error_code,omitempty"`
	ErrorMessage string               `json:"error_message,omitempty"`
	Lifecycle    EvaluationLifecycle  `json:"lifecycle"`
	Provenance   EvaluationProvenance `json:"provenance"`
	Released     bool                 `json:"released"`
}

// EvaluationInspector observes provider, toolkit, GPU, and CDI state without
// mutating the selected runtime.
type EvaluationInspector interface {
	Inspect(context.Context, EvaluationRequest) (EvaluationInspection, error)
}

// EvaluationCommandRunner is the environment-aware form of gpu.ExecRunner.
type EvaluationCommandRunner interface {
	RunWithEnv(context.Context, map[string]string, string, ...string) (gpu.CommandResult, error)
}

// EvaluationReservations is satisfied by gpu.ReservationStore.
type EvaluationReservations interface {
	Reserve(context.Context, []gpu.UUID, string, time.Duration) (gpu.ReservationLease, error)
	Release(context.Context, gpu.ReservationLease) error
}

// EvaluationAction is a distinct bounded host capability. It intentionally
// does not implement BackendDispatcher or enable Podman lifecycle deployment.
type EvaluationAction struct {
	Registry       *nvmsruntime.BackendRegistry
	Inventory      gpu.InventoryProvider
	Inspector      EvaluationInspector
	Runner         EvaluationCommandRunner
	Reservations   EvaluationReservations
	ReservationTTL time.Duration
	Filesystem     EvaluationFilesystem
}

// EvaluationError carries a stable machine-readable failure code.
type EvaluationError struct {
	Code string
	Err  error
}

func (e *EvaluationError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *EvaluationError) Unwrap() error { return e.Err }

func evaluationError(code, format string, values ...any) error {
	return &EvaluationError{Code: code, Err: fmt.Errorf(format, values...)}
}

// Execute validates, inspects, reserves, runs, verifies output, and releases.
func (action *EvaluationAction) Execute(ctx context.Context, request EvaluationRequest) (result EvaluationResult, returnedErr error) {
	result = newEvaluationResult(request)
	attachRequestProvenance(&result, request)
	if err := action.validate(request); err != nil {
		return failEvaluation(result, err)
	}

	filesystem := action.filesystem()
	if err := prepareOutputRootParent(filesystem, request.OutputRoot); err != nil {
		return failEvaluation(result, err)
	}
	coordinator, err := lockOutputRootCoordinator(ctx, filesystem, request.OutputRoot)
	if err != nil {
		return failEvaluation(result, evaluationError(CodeOutputLockFailed, "coordinate output_root lifecycle: %v", err))
	}
	defer func() {
		coordinator.releaseFile()
		coordinator.releaseProcess()
	}()

	root, err := prepareOutputRoot(filesystem, request.OutputRoot, true)
	if err != nil {
		return failEvaluation(result, err)
	}
	result.Provenance.OutputRootCreated = root.created
	availableBytes, err := filesystem.AvailableSpace(root.path)
	if err != nil {
		return failPreparedOutputRoot(filesystem, root, nil, result,
			evaluationError(CodeOutputRootSpaceFailed, "inspect output_root available space: %v", err))
	}
	result.Provenance.OutputRootAvailableBytes = &availableBytes

	outputLock, err := lockOutputRoot(ctx, filesystem, root.path)
	if err != nil {
		return failPreparedOutputRoot(filesystem, root, outputLock.ownedIdentity, result,
			evaluationError(CodeOutputLockFailed, "%v", err))
	}
	actionStarted := false
	defer func() {
		outputLock.releaseFile()
		defer outputLock.releaseProcess()
		if !root.created || actionStarted || returnedErr == nil {
			return
		}
		if cleanupErr := cleanupCreatedOutputRoot(filesystem, root, outputLock.ownedIdentity); cleanupErr != nil {
			result.Provenance.OutputRootCleanup = "preserved"
			result, returnedErr = failEvaluation(result, evaluationError(
				CodeOutputRootCleanupFailed,
				"preserve output_root after %v: %v",
				returnedErr,
				cleanupErr,
			))
			return
		}
		result.Provenance.OutputRootCleanup = "removed"
		result.Provenance.OutputRootCreated = false
	}()

	devices, executionInventory, err := action.resolveInventory(ctx, request)
	if err != nil {
		return failEvaluation(result, err)
	}

	uuids := requestedUUIDs(request.GPUBindings)
	lease, err := action.Reservations.Reserve(ctx, uuids, request.ManifestSHA256, action.ReservationTTL)
	if err != nil {
		return failEvaluation(result, evaluationError(CodeReservationFailed, "%v", err))
	}
	result.Released = false
	result.Provenance.ReservationToken = lease.Token
	if !lease.ExpiresAt.IsZero() {
		result.Provenance.ReservationExpiresAt = lease.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	defer func() {
		releaseCtx, cancelRelease := context.WithTimeout(context.Background(), EvaluationCleanupTimeout)
		defer cancelRelease()
		releaseErr := action.Reservations.Release(releaseCtx, lease)
		if releaseErr != nil {
			result.Success = false
			result.Released = false
			if result.ErrorCode == "" {
				result.ErrorCode = CodeCleanupFailed
				result.ErrorMessage = releaseErr.Error()
				returnedErr = evaluationError(CodeCleanupFailed, "release GPU reservation: %v", releaseErr)
				return
			}
			result.ErrorMessage = fmt.Sprintf("%s; cleanup_failed: release GPU reservation: %v", result.ErrorMessage, releaseErr)
			if returnedErr != nil {
				returnedErr = evaluationError(result.ErrorCode, "%v; cleanup_failed: release GPU reservation: %v", returnedErr, releaseErr)
			} else {
				returnedErr = evaluationError(result.ErrorCode, "%s", result.ErrorMessage)
			}
			return
		}
		result.Released = true
	}()

	before, err := snapshotJobDirectories(request.OutputRoot)
	if err != nil {
		return failEvaluation(result, evaluationError(CodeOutputSnapshotFailed, "snapshot output root: %v", err))
	}
	var inspection EvaluationInspection
	if executionInventory != nil {
		if inspector, ok := action.Inspector.(interface {
			InspectWithInventory(context.Context, EvaluationRequest, *gpu.ScopedInventory) (EvaluationInspection, error)
		}); ok {
			inspection, err = inspector.InspectWithInventory(ctx, request, executionInventory)
		} else {
			inspection, err = action.Inspector.Inspect(ctx, request)
		}
	} else {
		inspection, err = action.Inspector.Inspect(ctx, request)
	}
	if err != nil {
		var inspectionErr *EvaluationError
		if errors.As(err, &inspectionErr) {
			return failEvaluation(result, err)
		}
		return failEvaluation(result, evaluationError(CodeInspectionFailed, "%v", err))
	}
	result.Provenance.ToolkitRoot = inspection.ToolkitRoot
	result.Provenance.ToolkitExecutable = inspection.ToolkitExecutable
	result.Provenance.ToolkitVersion = inspection.Toolkit
	if err := validateInspection(request, inspection, devices); err != nil {
		return failEvaluation(result, err)
	}

	environment := cloneEnvironment(request.Environment)
	environment["DOCKER_HOST"] = request.PodmanPipe
	command := request.Executable
	arguments := append([]string(nil), request.Argv...)
	if request.WSLDistribution != "" {
		arguments = append([]string{"-d", request.WSLDistribution, "--", request.Executable}, arguments...)
		command = "wsl.exe"
	}
	result.Provenance.CommandSHA256 = commandFingerprint(command, arguments)
	actionStarted = true
	commandResult, runErr := action.Runner.RunWithEnv(ctx, environment, command, arguments...)
	result.Lifecycle = lifecycleFromCommand(commandResult)

	after, snapshotErr := snapshotJobDirectories(request.OutputRoot)
	jobDirectory := ""
	var outputErr error
	if snapshotErr == nil {
		var candidates []string
		jobDirectory, candidates, outputErr = validateNewJob(request, before, after)
		if len(candidates) > 0 {
			result.Provenance.CandidateJobDirectories = candidates
		}
		if jobDirectory != "" {
			result.Provenance.JobDirectory = jobDirectory
		}
	}

	// Prefer command outcome codes over job-output codes when the command failed,
	// while still attaching job-directory evidence gathered above.
	if commandResult.TimedOut || errors.Is(runErr, context.DeadlineExceeded) {
		return failEvaluation(result, evaluationError(CodeActionTimeout, "evaluation command exceeded %dms", request.TimeoutMillis))
	}
	if commandResult.Truncated {
		return failEvaluation(result, evaluationError(CodeOutputTruncated, "evaluation command exceeded %d-byte output bound", request.MaxOutputBytes))
	}
	if runErr != nil || commandResult.ExitCode != 0 {
		return failEvaluation(result, evaluationError(CodeActionFailed, "evaluation command exited %d: %v", commandResult.ExitCode, runErr))
	}
	if snapshotErr != nil {
		return failEvaluation(result, evaluationError(CodeOutputSnapshotFailed, "snapshot output root after action: %v", snapshotErr))
	}
	if outputErr != nil {
		return failEvaluation(result, outputErr)
	}
	result.Success = true
	return result, nil
}

func (action *EvaluationAction) validate(request EvaluationRequest) error {
	if action == nil || action.Registry == nil || action.Inventory == nil || action.Inspector == nil || action.Runner == nil || action.Reservations == nil {
		return evaluationError(CodeAdapterUnconfigured, "registry, inventory, inspector, runner, and reservations are required")
	}
	if err := validateOutputRootPath(request.OutputRoot); err != nil {
		return err
	}
	metadata, err := action.Registry.Resolve(request.Backend)
	if err != nil {
		return evaluationError(CodeProviderRejected, "%v", err)
	}
	if metadata.ID != nvmsruntime.BackendPodman || request.Backend != nvmsruntime.BackendPodman {
		return evaluationError(CodeProviderRejected, "evaluation requires backend %q exactly", nvmsruntime.BackendPodman)
	}
	if len(request.FallbackBackends) != 0 {
		return evaluationError(CodeFallbackRejected, "evaluation fallback is disabled")
	}
	if request.Version != EvaluationActionVersion {
		return evaluationError(CodeInvalidRequest, "unsupported evaluation action version %q", request.Version)
	}
	digest, err := hex.DecodeString(request.ManifestSHA256)
	if err != nil || len(digest) != sha256.Size {
		return evaluationError(CodeInvalidManifestDigest, "manifest_sha256 must contain exactly 64 hexadecimal characters")
	}
	if strings.TrimSpace(request.Executable) == "" || strings.ContainsRune(request.Executable, '\x00') {
		return evaluationError(CodeInvalidRequest, "executable is required")
	}
	if request.ExternalEngineToken != ExternalEngineDocker {
		return evaluationError(CodeInvalidEngineToken, "external_engine_token must remain %q", ExternalEngineDocker)
	}
	if !strings.HasPrefix(request.PodmanPipe, "npipe:////./pipe/") || strings.TrimPrefix(request.PodmanPipe, "npipe:////./pipe/") == "" {
		return evaluationError(CodeInvalidPodmanPipe, "a concrete Podman named pipe is required")
	}
	if err := validateManagedAbsolutePath(request.ReservationPath, "reservation_path", CodeInvalidReservationPath); err != nil {
		return err
	}
	if binder, ok := action.Reservations.(interface{ BoundReservationPath() string }); ok {
		bound := filepath.Clean(binder.BoundReservationPath())
		if bound != filepath.Clean(request.ReservationPath) {
			return evaluationError(CodeInvalidReservationPath, "reservation store path %q does not match request reservation_path", bound)
		}
	}
	timeout := time.Duration(request.TimeoutMillis) * time.Millisecond
	if request.TimeoutMillis <= 0 || timeout > MaxEvaluationTimeout {
		return evaluationError(CodeInvalidLimits, "timeout_millis must be in (0,%d]", MaxEvaluationTimeout.Milliseconds())
	}
	if request.MaxOutputBytes <= 0 || request.MaxOutputBytes > MaxEvaluationOutput {
		return evaluationError(CodeInvalidLimits, "max_output_bytes must be in (0,%d]", MaxEvaluationOutput)
	}
	minimumTTL := timeout + EvaluationReservationSkew
	if minimumTTL > MaxEvaluationTimeout {
		minimumTTL = MaxEvaluationTimeout
	}
	if action.ReservationTTL < minimumTTL || action.ReservationTTL > MaxEvaluationTimeout {
		return evaluationError(CodeInvalidLimits, "reservation TTL must cover command timeout plus %s skew and not exceed %s", EvaluationReservationSkew, MaxEvaluationTimeout)
	}
	if len(request.LockInvocation) == 0 {
		return evaluationError(CodeInvalidRequest, "lock_invocation is required")
	}
	if err := validateGPUBindings(request); err != nil {
		return err
	}
	return nil
}

func (action *EvaluationAction) resolveInventory(ctx context.Context, request EvaluationRequest) (map[gpu.UUID]gpu.Device, *gpu.ScopedInventory, error) {
	var (
		inventory []gpu.Device
		scoped    []gpu.ScopedInventory
		err       error
	)
	if provider, ok := action.Inventory.(interface {
		InventoryWithScopes(context.Context) ([]gpu.Device, []gpu.ScopedInventory, error)
	}); ok {
		inventory, scoped, err = provider.InventoryWithScopes(ctx)
	} else {
		inventory, err = action.Inventory.Inventory(ctx)
	}
	if err != nil {
		return nil, nil, evaluationError(CodeInventoryUnavailable, "%v", err)
	}
	byUUID := make(map[gpu.UUID]gpu.Device, len(inventory))
	for _, device := range inventory {
		if err := device.UUID.Validate(); err != nil {
			return nil, nil, evaluationError(CodeInventoryUnavailable, "inventory returned invalid UUID: %v", err)
		}
		if _, duplicate := byUUID[device.UUID]; duplicate {
			return nil, nil, evaluationError(CodeInventoryUnavailable, "inventory returned duplicate GPU UUID %q", device.UUID)
		}
		byUUID[device.UUID] = device
	}
	for _, declared := range request.ResourceManifest.GPUs {
		observed, exists := byUUID[declared.UUID]
		if !exists {
			return nil, nil, evaluationError(CodeInventoryMismatch, "GPU %q is absent from NanoVMS inventory", declared.UUID)
		}
		if err := validateInventoryScope(observed, request.WSLDistribution); err != nil {
			return nil, nil, evaluationError(CodeInventoryMismatch, "%v", err)
		}
		if err := verifyDeclaredGPUFacts(declared, observed); err != nil {
			return nil, nil, evaluationError(CodeInventoryMismatch, "%v", err)
		}
		if strings.TrimSpace(observed.Architecture) == "" || strings.TrimSpace(observed.ComputeCapability) == "" {
			return nil, nil, evaluationError(CodeInventoryUnavailable, "GPU %q inventory lacks authoritative architecture or compute capability", declared.UUID)
		}
		if err := gpu.ValidateCompatibility(observed, request.ResourceManifest.Artifact); err != nil {
			return nil, nil, evaluationError(CodeToolkitRejected, "%v", err)
		}
	}
	var executionInventory *gpu.ScopedInventory
	if len(scoped) > 0 {
		executionInventory, err = gpu.ExecutionScopedInventory(scoped, request.WSLDistribution)
		if err != nil {
			return nil, nil, evaluationError(CodeInventoryUnavailable, "%v", err)
		}
	}
	return byUUID, executionInventory, nil
}

func validateInventoryScope(device gpu.Device, distribution string) error {
	hostVisible := false
	executionVisible := distribution == ""
	for _, observation := range device.Observations {
		if observation.Scope == gpu.ScopeWindowsHost {
			hostVisible = true
		}
		if distribution != "" && observation.Scope == gpu.ScopeWSLDistro && observation.ScopeID == distribution {
			executionVisible = true
		}
	}
	if !hostVisible {
		return fmt.Errorf("GPU %q is absent from Windows host inventory", device.UUID)
	}
	if !executionVisible {
		return fmt.Errorf("GPU %q is absent from WSL distribution %q inventory", device.UUID, distribution)
	}
	return nil
}

func verifyDeclaredGPUFacts(declared, observed gpu.Device) error {
	facts := []struct {
		name     string
		declared string
		observed string
	}{
		{"name", declared.Name, observed.Name},
		{"architecture", declared.Architecture, observed.Architecture},
		{"compute capability", declared.ComputeCapability, observed.ComputeCapability},
		{"driver version", declared.DriverVersion, observed.DriverVersion},
		{"driver CUDA ceiling", declared.DriverCUDACeiling, observed.DriverCUDACeiling},
	}
	for _, fact := range facts {
		if strings.TrimSpace(fact.declared) != "" && !strings.EqualFold(strings.TrimSpace(fact.declared), strings.TrimSpace(fact.observed)) {
			return fmt.Errorf("GPU %q declared %s %q does not match observed %q", declared.UUID, fact.name, fact.declared, fact.observed)
		}
	}
	return nil
}

func validateGPUBindings(request EvaluationRequest) error {
	if len(request.GPUBindings) == 0 || len(request.ResourceManifest.GPUs) == 0 {
		return evaluationError(CodeGPUBindingRejected, "at least one GPU binding and manifest GPU are required")
	}
	if strings.TrimSpace(request.ResourceManifest.Artifact.CUDAToolkit) == "" {
		return evaluationError(CodeToolkitRejected, "artifact toolkit binding is required")
	}
	if _, err := request.ResourceManifest.CanonicalJSON(); err != nil {
		return evaluationError(CodeGPUBindingRejected, "invalid resource manifest: %v", err)
	}
	manifestUUIDs := make(map[gpu.UUID]struct{}, len(request.ResourceManifest.GPUs))
	for _, device := range request.ResourceManifest.GPUs {
		manifestUUIDs[device.UUID] = struct{}{}
	}
	seen := make(map[gpu.UUID]struct{}, len(request.GPUBindings))
	for _, binding := range request.GPUBindings {
		if err := binding.UUID.Validate(); err != nil {
			return evaluationError(CodeGPUBindingRejected, "%v", err)
		}
		if _, duplicate := seen[binding.UUID]; duplicate {
			return evaluationError(CodeGPUBindingRejected, "duplicate GPU binding %q", binding.UUID)
		}
		seen[binding.UUID] = struct{}{}
		if _, exists := manifestUUIDs[binding.UUID]; !exists {
			return evaluationError(CodeGPUBindingRejected, "GPU %q is absent from resource manifest", binding.UUID)
		}
		if binding.CUDAToolkit != request.ResourceManifest.Artifact.CUDAToolkit {
			return evaluationError(CodeToolkitRejected, "GPU %q toolkit does not match artifact toolkit", binding.UUID)
		}
		expectedCDI := "nvidia.com/gpu=" + string(binding.UUID)
		if binding.CDIDevice != expectedCDI {
			return evaluationError(CodeGPUBindingRejected, "GPU %q CDI binding must be %q", binding.UUID, expectedCDI)
		}
	}
	if len(seen) != len(manifestUUIDs) {
		return evaluationError(CodeGPUBindingRejected, "GPU bindings and resource manifest must contain the same UUIDs")
	}
	return nil
}

func validateInspection(request EvaluationRequest, inspection EvaluationInspection, inventory map[gpu.UUID]gpu.Device) error {
	if inspection.Provider != nvmsruntime.BackendPodman || inspection.PodmanPipe != request.PodmanPipe {
		return evaluationError(CodeInspectionMismatch, "inspected provider or Podman pipe does not match request")
	}
	if inspection.Toolkit != request.ResourceManifest.Artifact.CUDAToolkit {
		return evaluationError(CodeToolkitRejected, "installed toolkit %q does not match requested toolkit %q", inspection.Toolkit, request.ResourceManifest.Artifact.CUDAToolkit)
	}
	if !request.ResourceManifest.Artifact.CompiledKernels &&
		(strings.TrimSpace(inspection.ToolkitRoot) == "" || strings.TrimSpace(inspection.ToolkitExecutable) == "") {
		return evaluationError(CodeToolkitNotFound, "compiler-required artifact lacks resolved toolkit root or executable")
	}
	devices := make(map[gpu.UUID]gpu.Device, len(inspection.Devices))
	for _, device := range inspection.Devices {
		if err := device.UUID.Validate(); err != nil {
			return evaluationError(CodeInspectionMismatch, "inspection returned invalid UUID: %v", err)
		}
		devices[device.UUID] = device
	}
	for _, binding := range request.GPUBindings {
		device, exists := devices[binding.UUID]
		if !exists {
			return evaluationError(CodeInspectionMismatch, "GPU %q is absent from inspection", binding.UUID)
		}
		if _, exists := inventory[device.UUID]; !exists {
			return evaluationError(CodeInspectionMismatch, "GPU %q is absent from NanoVMS inventory", binding.UUID)
		}
		if inspection.CDIDevices[binding.UUID] != binding.CDIDevice {
			return evaluationError(CodeInspectionMismatch, "CDI inspection mismatch for GPU %q", binding.UUID)
		}
	}
	return nil
}

func newEvaluationResult(request EvaluationRequest) EvaluationResult {
	uuids := requestedUUIDs(request.GPUBindings)
	toolkitVersion := ""
	if request.ResourceManifest.Artifact.CompiledKernels {
		toolkitVersion = strings.TrimSpace(request.ResourceManifest.Artifact.CUDAToolkit)
	}
	return EvaluationResult{
		Version:   EvaluationActionVersion,
		Lifecycle: EvaluationLifecycle{ExitCode: -1},
		Released:  true,
		Provenance: EvaluationProvenance{
			ManifestSHA256:   strings.ToLower(request.ManifestSHA256),
			EffectiveEngine:  EffectiveEnginePodman,
			ResolvedProvider: nvmsruntime.BackendPodman,
			ExecutionPlane:   ExecutionPlaneNanoVMS,
			PodmanPipe:       request.PodmanPipe,
			GPUUUIDs:         uuids,
			ToolkitVersion:   toolkitVersion,
		},
	}
}

func attachRequestProvenance(result *EvaluationResult, request EvaluationRequest) {
	root := filepath.Clean(request.OutputRoot)
	result.Provenance.OutputRoot = root
	if root != "" && root != "." {
		result.Provenance.OutputLockPath = filepath.Join(root, outputRootLockFilename)
		result.Provenance.CoordinatorLockPath = filepath.Join(filepath.Dir(root), "."+filepath.Base(root)+outputRootCoordinatorSuffix)
	}
	if strings.TrimSpace(request.ReservationPath) != "" {
		result.Provenance.ReservationPath = filepath.Clean(request.ReservationPath)
	}
	if len(request.LockInvocation) > 0 {
		result.Provenance.LockInvocation = append([]string(nil), request.LockInvocation...)
	}
}

func commandFingerprint(executable string, argv []string) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(executable))
	_, _ = hasher.Write([]byte{0})
	for _, arg := range argv {
		_, _ = hasher.Write([]byte(arg))
		_, _ = hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func lifecycleFromCommand(command gpu.CommandResult) EvaluationLifecycle {
	stdoutBytes := bytes.ToValidUTF8(command.Stdout, []byte{0xEF, 0xBF, 0xBD})
	stderrBytes := bytes.ToValidUTF8(command.Stderr, []byte{0xEF, 0xBF, 0xBD})
	stdout := string(stdoutBytes)
	stderr := string(stderrBytes)
	stdoutHash := sha256.Sum256(stdoutBytes)
	stderrHash := sha256.Sum256(stderrBytes)
	return EvaluationLifecycle{
		ExitCode: command.ExitCode, DurationMS: command.Duration.Milliseconds(),
		TimedOut: command.TimedOut, Truncated: command.Truncated,
		Stdout: stdout, Stderr: stderr,
		StdoutSHA256: hex.EncodeToString(stdoutHash[:]), StderrSHA256: hex.EncodeToString(stderrHash[:]),
	}
}

func failEvaluation(result EvaluationResult, err error) (EvaluationResult, error) {
	result.Success = false
	var actionErr *EvaluationError
	if errors.As(err, &actionErr) {
		result.ErrorCode = actionErr.Code
		result.ErrorMessage = actionErr.Err.Error()
	} else {
		result.ErrorCode = CodeEvaluationFailed
		result.ErrorMessage = err.Error()
	}
	return result, err
}

func requestedUUIDs(bindings []EvaluationGPUBinding) []gpu.UUID {
	result := make([]gpu.UUID, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, binding.UUID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func cloneEnvironment(environment map[string]string) map[string]string {
	result := make(map[string]string, len(environment)+1)
	for key, value := range environment {
		result[key] = value
	}
	return result
}

func snapshotJobDirectories(root string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			result[entry.Name()] = struct{}{}
		}
	}
	return result, nil
}

func validateNewJob(request EvaluationRequest, before, after map[string]struct{}) (string, []string, error) {
	var added []string
	for name := range after {
		if _, existed := before[name]; !existed {
			added = append(added, name)
		}
	}
	sort.Strings(added)
	candidates := make([]string, 0, len(added))
	for _, name := range added {
		candidates = append(candidates, filepath.Join(request.OutputRoot, name))
	}
	if len(added) != 1 {
		return "", candidates, evaluationError(CodeAmbiguousJobOutput, "expected exactly one new job directory, found %d", len(added))
	}
	jobDirectory := candidates[0]
	data, err := os.ReadFile(filepath.Join(jobDirectory, JobLockFilename))
	if err != nil {
		return jobDirectory, candidates, evaluationError(CodeJobLockMismatch, "read job lock: %v", err)
	}
	var lock struct {
		Invocation []string `json:"invocation"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(&lock); err != nil {
		return jobDirectory, candidates, evaluationError(CodeJobLockMismatch, "decode job lock: %v", err)
	}
	if !lockInvocationsMatch(request.LockInvocation, lock.Invocation) {
		return jobDirectory, candidates, evaluationError(CodeJobLockMismatch, "job lock invocation does not match request")
	}
	return jobDirectory, candidates, nil
}

func lockInvocationsMatch(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for index := range expected {
		if !lockInvocationArgMatch(expected[index], actual[index], index == 0) {
			return false
		}
	}
	return true
}

func lockInvocationArgMatch(expected, actual string, executable bool) bool {
	if executable {
		return filepath.Base(expected) == filepath.Base(actual)
	}
	if strings.Contains(actual, "****") {
		return redactedAssignmentMatches(expected, actual)
	}
	return expected == actual
}

func redactedAssignmentMatches(expected, actual string) bool {
	expectedKey, expectedValue, expectedOK := envAssignment(expected)
	actualKey, actualValue, actualOK := envAssignment(actual)
	if expectedOK && actualOK && expectedKey == actualKey {
		return redactedValueMatches(expectedValue, actualValue)
	}
	return expected == actual
}

func envAssignment(value string) (string, string, bool) {
	key, remainder, ok := strings.Cut(value, "=")
	if !ok || key == "" {
		return "", "", false
	}
	return key, remainder, true
}

func redactedValueMatches(expected, actual string) bool {
	marker := "****"
	index := strings.Index(actual, marker)
	if index < 0 {
		return expected == actual
	}
	prefix := actual[:index]
	suffix := actual[index+len(marker):]
	return strings.HasPrefix(expected, prefix) && strings.HasSuffix(expected, suffix)
}

var outputRootLocks sync.Map

type acquiredOutputRootLock struct {
	file          *os.File
	mutex         *sync.Mutex
	ownedIdentity os.FileInfo
}

func (lock *acquiredOutputRootLock) releaseFile() {
	if lock.file != nil {
		unlockOutputFile(lock.file)
		lock.file = nil
	}
}

func (lock *acquiredOutputRootLock) releaseProcess() {
	if lock.mutex != nil {
		lock.mutex.Unlock()
		lock.mutex = nil
	}
}

func lockOutputRoot(ctx context.Context, filesystem EvaluationFilesystem, root string) (acquiredOutputRootLock, error) {
	return lockEvaluationPath(ctx, filesystem, filepath.Join(root, outputRootLockFilename))
}

func lockOutputRootCoordinator(ctx context.Context, filesystem EvaluationFilesystem, root string) (acquiredOutputRootLock, error) {
	filename := "." + filepath.Base(root) + outputRootCoordinatorSuffix
	return lockEvaluationPath(ctx, filesystem, filepath.Join(filepath.Dir(root), filename))
}

func lockEvaluationPath(ctx context.Context, filesystem EvaluationFilesystem, lockPath string) (acquiredOutputRootLock, error) {
	value, _ := outputRootLocks.LoadOrStore(lockPath, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	lock := acquiredOutputRootLock{mutex: mutex}
	lockFile, err := filesystem.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	created := err == nil
	if errors.Is(err, os.ErrExist) {
		lockFile, err = filesystem.OpenFile(lockPath, os.O_RDWR, 0o600)
	}
	if err != nil {
		mutex.Unlock()
		lock.mutex = nil
		return lock, fmt.Errorf("open output-root lock: %w", err)
	}
	lock.file = lockFile
	openedInfo, statErr := lockFile.Stat()
	pathInfo, lstatErr := filesystem.Lstat(lockPath)
	if created {
		lock.ownedIdentity = openedInfo
	}
	if statErr != nil || lstatErr != nil || isUnsafeOutputPathEntry(pathInfo) ||
		!pathInfo.Mode().IsRegular() || !os.SameFile(openedInfo, pathInfo) {
		ownedIdentity := lock.ownedIdentity
		lock.releaseFile()
		lock.releaseProcess()
		return acquiredOutputRootLock{ownedIdentity: ownedIdentity}, fmt.Errorf(
			"validate output-root lock: file stat %v, path stat %v",
			statErr,
			lstatErr,
		)
	}
	var lockWait *time.Timer
	defer func() {
		if lockWait != nil {
			lockWait.Stop()
		}
	}()
	for attempt := 0; ; attempt++ {
		acquired, lockErr := tryLockOutputFile(lockFile)
		if lockErr != nil {
			ownedIdentity := lock.ownedIdentity
			lock.releaseFile()
			lock.releaseProcess()
			return acquiredOutputRootLock{ownedIdentity: ownedIdentity}, fmt.Errorf("lock output root: %w", lockErr)
		}
		if acquired {
			return lock, nil
		}
		if err := waitForContendedLock(ctx, &lockWait, attempt); err != nil {
			ownedIdentity := lock.ownedIdentity
			lock.releaseFile()
			lock.releaseProcess()
			return acquiredOutputRootLock{ownedIdentity: ownedIdentity}, fmt.Errorf("lock output root: %w", err)
		}
	}
}

func waitForContendedLock(ctx context.Context, timer **time.Timer, attempt int) error {
	delays := [...]time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
	}
	delay := delays[len(delays)-1]
	if attempt >= 0 && attempt < len(delays) {
		delay = delays[attempt]
	}
	if *timer == nil {
		*timer = time.NewTimer(delay)
	} else {
		if !(*timer).Stop() {
			select {
			case <-(*timer).C:
			default:
			}
		}
		(*timer).Reset(delay)
	}
	select {
	case <-ctx.Done():
		if !(*timer).Stop() {
			<-(*timer).C
		}
		return ctx.Err()
	case <-(*timer).C:
		return nil
	}
}

func failPreparedOutputRoot(
	filesystem EvaluationFilesystem,
	root preparedOutputRoot,
	ownedLock os.FileInfo,
	result EvaluationResult,
	cause error,
) (EvaluationResult, error) {
	failedResult, returnedErr := failEvaluation(result, cause)
	if !root.created {
		return failedResult, returnedErr
	}
	if cleanupErr := cleanupCreatedOutputRoot(filesystem, root, ownedLock); cleanupErr != nil {
		failedResult.Provenance.OutputRootCleanup = "preserved"
		return failEvaluation(failedResult, evaluationError(
			CodeOutputRootCleanupFailed,
			"preserve output_root after %v: %v",
			cause,
			cleanupErr,
		))
	}
	failedResult.Provenance.OutputRootCleanup = "removed"
	failedResult.Provenance.OutputRootCreated = false
	return failedResult, returnedErr
}
