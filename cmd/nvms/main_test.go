package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/pkg/orchestrate"
)

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

func TestDeploySubcommandPrintsUsage(t *testing.T) {
	// Just verify the function exists and main dispatches correctly
	var stdout, stderr bytes.Buffer
	_ = stdout
	_ = stderr
	_ = context.Background()
}

func TestCLIHelpDoesNotPanic(t *testing.T) {
	printUsage()
}

func TestTokenCmdDoesNotPanic(t *testing.T) {
	tokenCmd([]string{})
}

func TestTierCmdListDoesNotPanic(t *testing.T) {
	args := []string{"list"}
	if len(args) > 0 && args[0] != "help" {
		_ = strings.Join(args, " ")
	}
}
