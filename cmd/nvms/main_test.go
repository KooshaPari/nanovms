package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kooshapari/nanovms/pkg/gpu"
	"github.com/kooshapari/nanovms/pkg/orchestrate"
)

func TestLifecycleCLIJSONBoundary(t *testing.T) {
	request := orchestrate.ServiceLifecycleRequest{Version: orchestrate.ServiceLifecycleVersion}
	input, _ := json.Marshal(request)
	var stdout, stderr bytes.Buffer
	execute := func(_ context.Context, got orchestrate.ServiceLifecycleRequest) (orchestrate.ServiceLifecycleResult, error) {
		if got.Version != request.Version {
			t.Fatalf("decoded request = %#v", got)
		}
		return orchestrate.ServiceLifecycleResult{Version: orchestrate.ServiceLifecycleVersion, Success: true}, nil
	}
	if exit := lifecycleCmd([]string{"--request", "-"}, bytes.NewReader(input), &stdout, &stderr, execute); exit != exitOK {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
	}
	var result orchestrate.ServiceLifecycleResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.Success {
		t.Fatalf("stdout is not one result: %q (%v)", stdout.String(), err)
	}
}

func TestActionCLIJSONBoundary(t *testing.T) {
	request := orchestrate.EvaluationRequest{Version: orchestrate.EvaluationActionVersion}
	input, _ := json.Marshal(request)
	var stdout, stderr bytes.Buffer
	execute := func(_ context.Context, got orchestrate.EvaluationRequest) (orchestrate.EvaluationResult, error) {
		if got.Version != request.Version {
			t.Fatalf("decoded request = %#v", got)
		}
		return orchestrate.EvaluationResult{Version: orchestrate.EvaluationActionVersion, Success: true}, nil
	}
	if exit := actionCmd([]string{"--request", "-"}, bytes.NewReader(input), &stdout, &stderr, execute); exit != exitOK {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr.String())
	}
	var result orchestrate.EvaluationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.Success {
		t.Fatalf("stdout is not one result: %q (%v)", stdout.String(), err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestActionCLIRejectsMalformedOrMultipleJSON(t *testing.T) {
	for _, input := range []string{`{`, `{}` + "\n" + `{}`} {
		var stdout, stderr bytes.Buffer
		exit := actionCmd([]string{"--request", "-"}, strings.NewReader(input), &stdout, &stderr, nil)
		if exit != exitInvalidJSON || stdout.Len() != 0 || !strings.Contains(stderr.String(), "invalid_json") {
			t.Fatalf("input %q: exit=%d stdout=%q stderr=%q", input, exit, stdout.String(), stderr.String())
		}
	}
}

func TestActionCLIStableFailureExitAndStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	execute := func(context.Context, orchestrate.EvaluationRequest) (orchestrate.EvaluationResult, error) {
		return orchestrate.EvaluationResult{
			Version:   orchestrate.EvaluationActionVersion,
			ErrorCode: orchestrate.CodeInspectionMismatch,
		}, &orchestrate.EvaluationError{Code: orchestrate.CodeInspectionMismatch, Err: errors.New("CDI differs")}
	}
	exit := actionCmd([]string{"--request", "-"}, strings.NewReader(`{}`), &stdout, &stderr, execute)
	if exit != orchestrate.ExitHostProbe {
		t.Fatalf("exit = %d, want %d", exit, orchestrate.ExitHostProbe)
	}
	if !strings.Contains(stderr.String(), "nvms action: inspection_mismatch:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if strings.Count(strings.TrimSpace(stdout.String()), "\n") != 0 {
		t.Fatalf("expected one JSON line: %q", stdout.String())
	}
}

func TestProcessExitTaxonomy(t *testing.T) {
	cases := map[string]int{
		orchestrate.CodeInvalidRequest:    orchestrate.ExitInvalidRequest,
		orchestrate.CodeReservationFailed: orchestrate.ExitContention,
		orchestrate.CodeInspectionMismatch: orchestrate.ExitHostProbe,
		orchestrate.CodeActionTimeout:     orchestrate.ExitActionRuntime,
		orchestrate.CodeCleanupFailed:     orchestrate.ExitEvidence,
	}
	for code, want := range cases {
		if got := orchestrate.ProcessExitFor(code); got != want {
			t.Fatalf("ProcessExitFor(%q)=%d want %d", code, got, want)
		}
	}
}

func TestActionCLIRequiresStdinRequestFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := actionCmd(nil, strings.NewReader(`{}`), &stdout, &stderr, nil); exit != exitUsage {
		t.Fatalf("exit = %d", exit)
	}
}

func TestEvaluationInventoryProviderWiresWindowsAndSelectedWSL(t *testing.T) {
	runner := &gpu.ExecRunner{Timeout: time.Second, MaxOutput: 1024}
	native := evaluationInventoryProvider(orchestrate.EvaluationRequest{}, runner)
	if len(native.Adapters) != 1 {
		t.Fatalf("native adapters = %d, want Windows only", len(native.Adapters))
	}
	if _, ok := native.Adapters[0].(gpu.WindowsInventoryAdapter); !ok {
		t.Fatalf("native adapter = %T, want Windows inventory", native.Adapters[0])
	}

	const distribution = "Ubuntu-24.04"
	wsl := evaluationInventoryProvider(orchestrate.EvaluationRequest{WSLDistribution: distribution}, runner)
	if len(wsl.Adapters) != 2 {
		t.Fatalf("WSL adapters = %d, want Windows and WSL", len(wsl.Adapters))
	}
	adapter, ok := wsl.Adapters[1].(gpu.WSLInventoryAdapter)
	if !ok || adapter.Distribution != distribution {
		t.Fatalf("WSL adapter = %#v", wsl.Adapters[1])
	}
}
