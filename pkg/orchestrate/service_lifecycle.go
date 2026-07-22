// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

const (
	// ServiceLifecycleVersion is the NanoVMS service lifecycle wire contract.
	ServiceLifecycleVersion = "nanovms.io/service-lifecycle/v1"
	// PhenoComposeLifecycleSchema is the lifecycle plan schema emitted by PhenoCompose.
	PhenoComposeLifecycleSchema = "phenocompose.lifecycle/v0"
)

// ServiceLifecycleRequest is the stdin JSON contract for `nvms lifecycle --request -`.
type ServiceLifecycleRequest struct {
	Version          string                       `json:"version"`
	SchemaVersion    string                       `json:"schema_version"`
	ManifestSHA256   string                       `json:"manifest_sha256"`
	RunID            string                       `json:"run_id"`
	WSLDistribution  string                       `json:"wsl_distribution"`
	PodmanPipe       string                       `json:"podman_pipe"`
	Order            []string                     `json:"order"`
	Intents          []ServiceLifecycleIntent     `json:"intents"`
	Services         map[string]ServiceDefinition `json:"services"`
}

// ServiceLifecycleIntent mirrors PhenoCompose lifecycle intents.
type ServiceLifecycleIntent struct {
	Phase      string `json:"phase"`
	Service    string `json:"service"`
	Image      string `json:"image,omitempty"`
	DependsOn  []string `json:"depends_on,omitempty"`
}

// ServiceDefinition carries per-service spawn configuration.
type ServiceDefinition struct {
	Image       string            `json:"image"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	CPUMillis   *uint32           `json:"cpu_millis,omitempty"`
	MemoryBytes *uint64           `json:"memory_bytes,omitempty"`
	GPUUUIDs    []string          `json:"gpu_uuids,omitempty"`
}

// ServiceLifecycleResult records bounded lifecycle evidence for PhenoCompose.
type ServiceLifecycleResult struct {
	Version        string            `json:"version"`
	Success        bool              `json:"success"`
	ErrorCode      string            `json:"error_code,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	Containers     map[string]string `json:"containers,omitempty"`
	EffectiveEngine string           `json:"effective_engine"`
	ResolvedProvider string          `json:"resolved_provider"`
	PodmanPipe     string            `json:"podman_pipe"`
}

// ServiceLifecycleAction executes create-phase Podman lifecycle intents through
// the bounded command runner without routing through generic tier dispatchers.
type ServiceLifecycleAction struct {
	Runner gpu.CommandRunner
}

// Execute validates the request and runs podman create intents in plan order.
func (action ServiceLifecycleAction) Execute(ctx context.Context, request ServiceLifecycleRequest) (ServiceLifecycleResult, error) {
	result := ServiceLifecycleResult{
		Version:          ServiceLifecycleVersion,
		EffectiveEngine:  EffectiveEnginePodman,
		ResolvedProvider: EffectiveEnginePodman,
		PodmanPipe:       request.PodmanPipe,
		Containers:       map[string]string{},
	}
	if err := validateServiceLifecycleRequest(request); err != nil {
		return failServiceLifecycle(result, err)
	}
	if action.Runner == nil {
		return failServiceLifecycle(result, evaluationError(CodeInspectionFailed, "service lifecycle runner is not configured"))
	}
	inspector := HostEvaluationInspector{Runner: action.Runner}
	if _, err := inspector.runWithEnv(ctx, request.WSLDistribution, map[string]string{"DOCKER_HOST": request.PodmanPipe}, "podman", "info", "--format", "json"); err != nil {
		return failServiceLifecycle(result, evaluationError(CodeInspectionFailed, "Podman inspection failed: %v", err))
	}
	created := make([]string, 0)
	for _, intent := range request.Intents {
		if intent.Phase != "create" {
			continue
		}
		service, exists := request.Services[intent.Service]
		if !exists {
			action.rollback(ctx, request, created)
			return failServiceLifecycle(result, evaluationError(CodeInvalidRequest, "lifecycle intent references unknown service %q", intent.Service))
		}
		containerID, err := action.spawn(ctx, request, intent.Service, service)
		if err != nil {
			action.rollback(ctx, request, created)
			return failServiceLifecycle(result, err)
		}
		result.Containers[intent.Service] = containerID
		created = append(created, containerID)
	}
	result.Success = true
	return result, nil
}

func validateServiceLifecycleRequest(request ServiceLifecycleRequest) error {
	if request.Version != ServiceLifecycleVersion {
		return evaluationError(CodeInvalidRequest, "unsupported service lifecycle version %q", request.Version)
	}
	if request.SchemaVersion != PhenoComposeLifecycleSchema {
		return evaluationError(CodeInvalidRequest, "unsupported lifecycle schema %q", request.SchemaVersion)
	}
	if len(request.ManifestSHA256) != 64 {
		return evaluationError(CodeInvalidRequest, "manifest_sha256 must be a 64-character digest")
	}
	if strings.TrimSpace(request.RunID) == "" {
		return evaluationError(CodeInvalidRequest, "run_id must not be empty")
	}
	if strings.TrimSpace(request.PodmanPipe) == "" {
		return evaluationError(CodeInvalidPodmanPipe, "podman_pipe must not be empty")
	}
	if len(request.Intents) == 0 || len(request.Services) == 0 {
		return evaluationError(CodeInvalidRequest, "lifecycle plan must include intents and services")
	}
	return nil
}

func (action ServiceLifecycleAction) spawn(ctx context.Context, request ServiceLifecycleRequest, serviceName string, service ServiceDefinition) (string, error) {
	if strings.TrimSpace(service.Image) == "" {
		return "", evaluationError(CodeInvalidRequest, "service %q image must not be empty", serviceName)
	}
	name := fmt.Sprintf("%s-%s", request.RunID, serviceName)
	args := []string{"run", "--detach", "--name", name}
	for key, value := range service.Environment {
		args = append(args, "--env", key+"="+value)
	}
	if service.CPUMillis != nil {
		args = append(args, "--cpus", fmt.Sprintf("%.3f", float64(*service.CPUMillis)/1000.0))
	}
	if service.MemoryBytes != nil {
		args = append(args, "--memory", fmt.Sprintf("%d", *service.MemoryBytes))
	}
	for _, uuid := range service.GPUUUIDs {
		args = append(args, "--device", "nvidia.com/gpu="+uuid)
	}
	args = append(args, service.Image)
	args = append(args, service.Command...)
	inspector := HostEvaluationInspector{Runner: action.Runner}
	result, err := inspector.runWithEnv(ctx, request.WSLDistribution, map[string]string{"DOCKER_HOST": request.PodmanPipe}, "podman", args...)
	if err != nil {
		return "", evaluationError(CodeActionFailed, "podman run failed for %q: %v", serviceName, err)
	}
	if result.ExitCode != 0 {
		return "", evaluationError(CodeActionFailed, "podman run for %q exited %d: %s", serviceName, result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	id := strings.TrimSpace(string(result.Stdout))
	if id == "" {
		return "", evaluationError(CodeActionFailed, "podman run for %q succeeded without a container ID", serviceName)
	}
	return id, nil
}

func (action ServiceLifecycleAction) rollback(ctx context.Context, request ServiceLifecycleRequest, containerIDs []string) {
	inspector := HostEvaluationInspector{Runner: action.Runner}
	for index := len(containerIDs) - 1; index >= 0; index-- {
		_, _ = inspector.runWithEnv(ctx, request.WSLDistribution, map[string]string{"DOCKER_HOST": request.PodmanPipe}, "podman", "stop", containerIDs[index])
	}
}

func failServiceLifecycle(result ServiceLifecycleResult, err error) (ServiceLifecycleResult, error) {
	code := CodeEvaluationFailed
	message := err
	var evaluationErr *EvaluationError
	if errors.As(err, &evaluationErr) {
		code = evaluationErr.Code
		message = evaluationErr.Err
	}
	result.Success = false
	result.ErrorCode = code
	result.ErrorMessage = message.Error()
	return result, err
}
