// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"context"
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
)

var nvccReleasePattern = regexp.MustCompile(`(?i)\brelease\s+([0-9]+\.[0-9]+)\b`)

// HostToolkitInspector discovers installed CUDA toolkits using environment precedence.
type HostToolkitInspector struct {
	Runner             gpu.CommandRunner
	ToolkitRoot        string
	LookupEnv          func(string) (string, bool)
	ProgramFilesRoots  []string
	WSLToolkitProvider WSLToolkitProvider
}

// WSLToolkitProvider returns candidate toolkit roots for one selected distribution.
type WSLToolkitProvider interface {
	ToolkitRoots(context.Context, string, string) ([]string, error)
}

type defaultWSLToolkitProvider struct{}

func (defaultWSLToolkitProvider) ToolkitRoots(_ context.Context, _ string, version string) ([]string, error) {
	return []string{"/usr/local/cuda-" + version}, nil
}

// ResolvedToolkit records one discovered CUDA toolkit installation.
type ResolvedToolkit struct {
	Root       string `json:"root"`
	Executable string `json:"executable"`
	Version    string `json:"version"`
}

// ToolkitDiscoveryRequest scopes one bounded toolkit discovery probe.
type ToolkitDiscoveryRequest struct {
	WSLDistribution string
	Environment     map[string]string
	CUDAToolkit     string
}

type resolvedToolkit struct {
	Root       string
	Executable string
	Version    string
}

type toolkitCandidateGroup struct {
	roots         []string
	authoritative bool
}

type inspectionEnvRunner interface {
	RunWithEnv(context.Context, map[string]string, string, ...string) (gpu.CommandResult, error)
}

// ResolveToolkit discovers an installed CUDA toolkit using environment precedence.
func (inspector HostToolkitInspector) ResolveToolkit(ctx context.Context, request ToolkitDiscoveryRequest) (ResolvedToolkit, error) {
	resolved, err := inspector.resolveToolkit(ctx, request)
	if err != nil {
		return ResolvedToolkit{}, err
	}
	return ResolvedToolkit(resolved), nil
}

func (inspector HostToolkitInspector) resolveToolkit(ctx context.Context, request ToolkitDiscoveryRequest) (resolvedToolkit, error) {
	requested := strings.TrimSpace(request.CUDAToolkit)
	if requested == "" {
		return resolvedToolkit{}, providerError(CodeToolkitNotFound, "requested CUDA toolkit version is empty")
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
			return resolvedToolkit{}, providerError(CodeToolkitAmbiguous,
				"multiple CUDA %s installations found: %s", requested, strings.Join(locations, ", "))
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if group.authoritative {
			if len(mismatches) != 0 {
				return resolvedToolkit{}, providerError(CodeToolkitVersionMismatch,
					"configured CUDA toolkit does not match requested %s: %s", requested, strings.Join(mismatches, "; "))
			}
			if groupUnavailable {
				return resolvedToolkit{}, providerError(CodeToolkitNotFound,
					"configured CUDA %s toolkit has no executable nvcc", requested)
			}
		}
	}
	if len(mismatches) != 0 {
		return resolvedToolkit{}, providerError(CodeToolkitVersionMismatch,
			"no CUDA toolkit matched requested %s: %s", requested, strings.Join(mismatches, "; "))
	}
	return resolvedToolkit{}, providerError(CodeToolkitNotFound, "CUDA toolkit %s was not found", requested)
}

func (inspector HostToolkitInspector) toolkitCandidateGroups(
	ctx context.Context,
	request ToolkitDiscoveryRequest,
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
				return "", providerError(CodeToolkitAmbiguous,
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
				return nil, providerError(CodeToolkitInspectionTimeout,
					"CUDA toolkit discovery timed out in WSL distribution %q", request.WSLDistribution)
			}
			return nil, providerError(CodeToolkitNotFound, "discover CUDA toolkit in WSL distribution %q: %v", request.WSLDistribution, err)
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

func (inspector HostToolkitInspector) inspectToolkitCandidate(
	ctx context.Context,
	distribution string,
	root string,
) (resolvedToolkit, string, bool, error) {
	hostDistribution := distribution
	absolute := filepath.IsAbs(root)
	if distribution != "" {
		if absolute {
			// Configured Windows toolkit roots stay on the host even when
			// discovery is scoped to a WSL distribution.
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
		return candidate, "", false, providerError(CodeToolkitInspectionTimeout,
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

func (inspector HostToolkitInspector) run(ctx context.Context, distribution, executable string, arguments ...string) (gpu.CommandResult, error) {
	return inspector.runWithEnv(ctx, distribution, nil, executable, arguments...)
}

func (inspector HostToolkitInspector) runWithEnv(
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
