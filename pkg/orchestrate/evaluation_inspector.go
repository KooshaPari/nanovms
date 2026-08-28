// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/kooshapari/nanovms/pkg/gpu"
	nvmsruntime "github.com/kooshapari/nanovms/pkg/runtime"
)

var nvccReleasePattern = regexp.MustCompile(`(?i)\brelease\s+([0-9]+\.[0-9]+)\b`)

// HostEvaluationInspector performs bounded, read-only host probes. When a WSL
// distribution is selected every probe is routed through that distribution.
type HostEvaluationInspector struct {
	Runner             gpu.CommandRunner
	ToolkitRoot        string
	LookupEnv          func(string) (string, bool)
	ProgramFilesRoots  []string
	WSLToolkitProvider WSLToolkitProvider
}

// WSLToolkitProvider returns candidate toolkit roots for one selected
// distribution. The inspector probes every returned executable only through
// its injected command runner.
type WSLToolkitProvider interface {
	ToolkitRoots(context.Context, string, string) ([]string, error)
}

type defaultWSLToolkitProvider struct{}

func (defaultWSLToolkitProvider) ToolkitRoots(_ context.Context, _ string, version string) ([]string, error) {
	return []string{"/usr/local/cuda-" + version}, nil
}

type resolvedToolkit struct {
	Root       string
	Executable string
	Version    string
}

// ResolvedToolkit records one discovered CUDA toolkit installation.
type ResolvedToolkit struct {
	Root       string `json:"root"`
	Executable string `json:"executable"`
	Version    string `json:"version"`
}

// ResolveToolkit discovers an installed CUDA toolkit using evaluation precedence.
func (inspector HostEvaluationInspector) ResolveToolkit(ctx context.Context, request EvaluationRequest) (ResolvedToolkit, error) {
	resolved, err := inspector.resolveToolkit(ctx, request)
	if err != nil {
		return ResolvedToolkit{}, err
	}
	return ResolvedToolkit(resolved), nil
}

type toolkitCandidateGroup struct {
	roots         []string
	authoritative bool
}

// Inspect verifies Podman responsiveness and independently observes GPU UUIDs
// and CDI device names. Compiler-required artifacts additionally require an
// installed nvcc toolkit; precompiled artifacts preserve the requested toolkit
// version without requiring a toolkit root or executable.
type inspectionEnvRunner interface {
	RunWithEnv(context.Context, map[string]string, string, ...string) (gpu.CommandResult, error)
}

func (inspector HostEvaluationInspector) Inspect(ctx context.Context, request EvaluationRequest) (EvaluationInspection, error) {
	return inspector.InspectWithInventory(ctx, request, nil)
}

// InspectWithInventory performs host probes, reusing executionInventory when provided
// so callers can avoid a duplicate nvidia-smi query after inventory reconciliation.
func (inspector HostEvaluationInspector) InspectWithInventory(
	ctx context.Context,
	request EvaluationRequest,
	executionInventory *gpu.ScopedInventory,
) (EvaluationInspection, error) {
	if inspector.Runner == nil {
		return EvaluationInspection{}, fmt.Errorf("inspection command runner is required")
	}
	pipe := strings.TrimSpace(request.PodmanPipe)
	if pipe == "" {
		return EvaluationInspection{}, fmt.Errorf("podman pipe is required for inspection")
	}
	info, err := inspector.runWithEnv(ctx, request.WSLDistribution, map[string]string{"DOCKER_HOST": pipe}, "podman", "info", "--format", "json")
	if err != nil {
		return EvaluationInspection{}, fmt.Errorf("podman inspection failed: %w", err)
	}
	if info.ExitCode != 0 {
		return EvaluationInspection{}, fmt.Errorf("podman inspection failed: podman info exited %d", info.ExitCode)
	}
	var infoDocument map[string]any
	if err := json.Unmarshal(info.Stdout, &infoDocument); err != nil || len(infoDocument) == 0 {
		return EvaluationInspection{}, fmt.Errorf("podman inspection returned malformed JSON")
	}

	var inventory gpu.ScopedInventory
	if executionInventory != nil {
		if err := validateExecutionInventoryScope(*executionInventory, request.WSLDistribution); err != nil {
			return EvaluationInspection{}, err
		}
		inventory = *executionInventory
	} else if request.WSLDistribution == "" {
		inventory, err = (gpu.WindowsInventoryAdapter{Runner: inspector.Runner}).Inventory(ctx)
		if err != nil {
			return EvaluationInspection{}, err
		}
	} else {
		inventory, err = (gpu.WSLInventoryAdapter{Runner: inspector.Runner, Distribution: request.WSLDistribution}).Inventory(ctx)
		if err != nil {
			return EvaluationInspection{}, err
		}
	}

	requestedToolkit := strings.TrimSpace(request.ResourceManifest.Artifact.CUDAToolkit)
	toolkit := resolvedToolkit{Version: requestedToolkit}
	if !request.ResourceManifest.Artifact.CompiledKernels {
		toolkit, err = inspector.resolveToolkit(ctx, request)
		if err != nil {
			return EvaluationInspection{}, err
		}
	}

	cdiList, err := inspector.runWithEnv(ctx, request.WSLDistribution, map[string]string{"DOCKER_HOST": pipe}, "nvidia-ctk", "cdi", "list")
	if err != nil {
		return EvaluationInspection{}, fmt.Errorf("CDI inspection failed: %w", err)
	}
	cdiDevices := expandCDIDevices(cdiList.Stdout, inventory)
	return EvaluationInspection{
		Provider: nvmsruntime.BackendPodman, PodmanPipe: request.PodmanPipe,
		Toolkit: toolkit.Version, ToolkitRoot: toolkit.Root, ToolkitExecutable: toolkit.Executable,
		Devices: inventory.Devices, CDIDevices: cdiDevices,
	}, nil
}

// expandCDIDevices parses nvidia-ctk cdi list output. WSL Podman machines often
// expose only nvidia.com/gpu=all; map each inventoried UUID to its canonical
// per-device CDI name so validation matches manifest bindings.
func expandCDIDevices(stdout []byte, inventory gpu.ScopedInventory) map[gpu.UUID]string {
	const prefix = "nvidia.com/gpu="
	devices := make(map[gpu.UUID]string)
	for _, field := range strings.Fields(string(stdout)) {
		if !strings.HasPrefix(field, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(field, prefix)
		if suffix == "all" {
			continue
		}
		uuid, parseErr := gpu.ParseUUID(suffix)
		if parseErr == nil {
			devices[uuid] = prefix + string(uuid)
		}
	}
	if strings.Contains(string(stdout), prefix+"all") {
		for _, device := range inventory.Devices {
			if err := device.UUID.Validate(); err != nil {
				continue
			}
			if _, exists := devices[device.UUID]; !exists {
				devices[device.UUID] = prefix + string(device.UUID)
			}
		}
	}
	return devices
}

func validateExecutionInventoryScope(inventory gpu.ScopedInventory, distribution string) error {
	if distribution == "" {
		if inventory.Scope != gpu.ScopeWindowsHost {
			return fmt.Errorf("execution inventory scope %q does not match native host execution", inventory.Scope)
		}
		return nil
	}
	if inventory.Scope != gpu.ScopeWSLDistro || inventory.ScopeID != distribution {
		return fmt.Errorf("execution inventory scope %q/%q does not match WSL distribution %q", inventory.Scope, inventory.ScopeID, distribution)
	}
	return nil
}

func (inspector HostEvaluationInspector) resolveToolkit(ctx context.Context, request EvaluationRequest) (resolvedToolkit, error) {
	requested := strings.TrimSpace(request.ResourceManifest.Artifact.CUDAToolkit)
	if requested == "" {
		return resolvedToolkit{}, evaluationError(CodeToolkitNotFound, "requested CUDA toolkit version is empty")
	}
	groups, err := inspector.toolkitCandidateGroups(ctx, request, requested)
	if err != nil {
		return resolvedToolkit{}, err
	}

	var mismatches []string
	for _, group := range groups {
		roots := uniqueNonempty(group.roots)
		if len(roots) == 0 {
			continue
		}
		var matches []resolvedToolkit
		groupUnavailable := false
		for _, root := range roots {
			candidate, observed, available, inspectErr := inspector.inspectToolkitCandidate(ctx, request.WSLDistribution, root)
			if inspectErr != nil {
				return resolvedToolkit{}, inspectErr
			}
			if !available {
				groupUnavailable = true
				continue
			}
			if observed != requested {
				mismatches = append(mismatches, fmt.Sprintf("%s reported %s", candidate.Executable, observed))
				continue
			}
			candidate.Version = observed
			matches = append(matches, candidate)
		}
		if len(matches) > 1 {
			locations := make([]string, 0, len(matches))
			for _, match := range matches {
				locations = append(locations, match.Root)
			}
			sort.Strings(locations)
			return resolvedToolkit{}, evaluationError(CodeToolkitAmbiguous,
				"multiple CUDA %s installations found: %s", requested, strings.Join(locations, ", "))
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if group.authoritative {
			if len(mismatches) != 0 {
				return resolvedToolkit{}, evaluationError(CodeToolkitVersionMismatch,
					"configured CUDA toolkit does not match requested %s: %s", requested, strings.Join(mismatches, "; "))
			}
			if groupUnavailable {
				return resolvedToolkit{}, evaluationError(CodeToolkitNotFound,
					"configured CUDA %s toolkit has no executable nvcc", requested)
			}
		}
	}
	if len(mismatches) != 0 {
		return resolvedToolkit{}, evaluationError(CodeToolkitVersionMismatch,
			"no CUDA toolkit matched requested %s: %s", requested, strings.Join(mismatches, "; "))
	}
	return resolvedToolkit{}, evaluationError(CodeToolkitNotFound, "CUDA toolkit %s was not found", requested)
}

func (inspector HostEvaluationInspector) toolkitCandidateGroups(
	ctx context.Context,
	request EvaluationRequest,
	version string,
) ([]toolkitCandidateGroup, error) {
	lookup := inspector.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	requestEnv := func(key string) (string, error) {
		var keys []string
		for candidate, value := range request.Environment {
			if strings.EqualFold(candidate, key) && strings.TrimSpace(value) != "" {
				keys = append(keys, candidate)
			}
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			return "", nil
		}
		value := strings.TrimSpace(request.Environment[keys[0]])
		for _, candidate := range keys[1:] {
			if strings.TrimSpace(request.Environment[candidate]) != value {
				return "", evaluationError(CodeToolkitAmbiguous,
					"environment contains conflicting case-insensitive values for %s", key)
			}
		}
		return value, nil
	}
	environmentValue := func(key string, allowHost bool) (string, error) {
		value, err := requestEnv(key)
		if err != nil || value != "" {
			return value, err
		}
		if !allowHost {
			return "", nil
		}
		value, _ = lookup(key)
		return strings.TrimSpace(value), nil
	}

	configured := strings.TrimSpace(inspector.ToolkitRoot)
	if configured == "" {
		var err error
		configured, err = environmentValue("NANOVMS_CUDA_TOOLKIT_ROOT", request.WSLDistribution == "")
		if err != nil {
			return nil, err
		}
	}
	versionEnvironment := "CUDA_PATH_V" + strings.NewReplacer(".", "_", "-", "_").Replace(version)
	versionRoot, err := environmentValue(versionEnvironment, request.WSLDistribution == "")
	if err != nil {
		return nil, err
	}
	cudaPath, err := environmentValue("CUDA_PATH", request.WSLDistribution == "")
	if err != nil {
		return nil, err
	}
	groups := []toolkitCandidateGroup{
		{roots: []string{configured}, authoritative: true},
		{roots: []string{versionRoot}, authoritative: true},
		{roots: []string{cudaPath}},
	}
	if request.WSLDistribution != "" {
		provider := inspector.WSLToolkitProvider
		if provider == nil {
			provider = defaultWSLToolkitProvider{}
		}
		roots, err := provider.ToolkitRoots(ctx, request.WSLDistribution, version)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, evaluationError(CodeToolkitInspectionTO,
					"CUDA toolkit discovery timed out in WSL distribution %q", request.WSLDistribution)
			}
			return nil, evaluationError(CodeToolkitNotFound, "discover CUDA toolkit in WSL distribution %q: %v", request.WSLDistribution, err)
		}
		return append(groups, toolkitCandidateGroup{roots: roots}), nil
	}

	programRoots := append([]string(nil), inspector.ProgramFilesRoots...)
	if programRoots == nil {
		for _, key := range []string{"ProgramFiles", "ProgramW6432"} {
			value, envErr := environmentValue(key, true)
			if envErr != nil {
				return nil, envErr
			}
			if value != "" {
				programRoots = append(programRoots, value)
			}
		}
		if len(programRoots) == 0 && runtime.GOOS == "windows" {
			programRoots = append(programRoots, `C:\Program Files`)
		}
	}
	standardRoots := make([]string, 0, len(programRoots))
	for _, root := range programRoots {
		standardRoots = append(standardRoots, filepath.Join(root, "NVIDIA GPU Computing Toolkit", "CUDA", "v"+version))
	}
	return append(groups, toolkitCandidateGroup{roots: standardRoots}), nil
}

func (inspector HostEvaluationInspector) inspectToolkitCandidate(
	ctx context.Context,
	distribution string,
	root string,
) (resolvedToolkit, string, bool, error) {
	hostDistribution := distribution
	absolute := filepath.IsAbs(root)
	if distribution != "" {
		if absolute {
			// Configured Windows toolkit roots stay on the host even when
			// evaluation targets a WSL distribution.
			hostDistribution = ""
		} else {
			absolute = path.IsAbs(root)
		}
	}
	if !absolute {
		return resolvedToolkit{Root: root}, "", false, nil
	}
	executable := filepath.Join(root, "bin", "nvcc.exe")
	if hostDistribution != "" {
		executable = path.Join(root, "bin", "nvcc")
	}
	candidate := resolvedToolkit{Root: root, Executable: executable}
	result, err := inspector.run(ctx, hostDistribution, executable, "--version")
	if result.TimedOut || errors.Is(err, context.DeadlineExceeded) {
		return candidate, "", false, evaluationError(CodeToolkitInspectionTO,
			"CUDA toolkit inspection timed out for %q", executable)
	}
	if err != nil || result.ExitCode != 0 {
		return candidate, "", false, nil
	}
	match := nvccReleasePattern.FindSubmatch(append(append([]byte(nil), result.Stdout...), result.Stderr...))
	if match == nil {
		return candidate, "unknown", true, nil
	}
	return candidate, string(match[1]), true, nil
}

func uniqueNonempty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := value
		if runtime.GOOS == "windows" {
			key = strings.ToLower(filepath.Clean(value))
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (inspector HostEvaluationInspector) run(ctx context.Context, distribution, executable string, arguments ...string) (gpu.CommandResult, error) {
	return inspector.runWithEnv(ctx, distribution, nil, executable, arguments...)
}

func (inspector HostEvaluationInspector) runWithEnv(
	ctx context.Context,
	distribution string,
	environment map[string]string,
	executable string,
	arguments ...string,
) (gpu.CommandResult, error) {
	command := executable
	args := arguments
	if distribution != "" {
		command = "wsl.exe"
		args = append([]string{"-d", distribution, "--", executable}, arguments...)
	}
	if len(environment) == 0 {
		return inspector.Runner.Run(ctx, command, args...)
	}
	runner, ok := inspector.Runner.(inspectionEnvRunner)
	if !ok {
		return gpu.CommandResult{}, fmt.Errorf("inspection command runner must support environment overrides")
	}
	return runner.RunWithEnv(ctx, environment, command, args...)
}
