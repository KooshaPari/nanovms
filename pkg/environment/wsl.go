// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

type canonicalWSLMetadata struct {
	Distribution string `json:"distribution"`
	Kernel       string `json:"kernel,omitempty"`
}

// InspectWSL reads one selected distribution name and kernel without mutating it.
func InspectWSL(ctx context.Context, runner gpu.CommandRunner, distribution string) (*WSLMetadata, error) {
	distribution = strings.TrimSpace(distribution)
	if distribution == "" {
		return nil, nil
	}
	if runner == nil {
		return nil, providerError(CodeWSLInspectionFailed, "command runner is required")
	}
	result, err := runner.Run(ctx, "wsl.exe", "-d", distribution, "--", "uname", "-r")
	if err != nil {
		return nil, providerError(CodeWSLInspectionFailed, "inspect WSL distribution %q kernel: %v", distribution, err)
	}
	if result.ExitCode != 0 {
		return nil, providerError(CodeWSLInspectionFailed, "inspect WSL distribution %q kernel exited %d", distribution, result.ExitCode)
	}
	kernel := strings.TrimSpace(string(result.Stdout))
	metadata := &WSLMetadata{Distribution: distribution, Kernel: kernel}
	digest, digestErr := wslMetadataDigest(*metadata)
	if digestErr != nil {
		return nil, digestErr
	}
	metadata.Digest = digest
	return metadata, nil
}

func wslMetadataDigest(metadata WSLMetadata) (string, error) {
	payload, err := json.Marshal(canonicalWSLMetadata{
		Distribution: metadata.Distribution,
		Kernel:       metadata.Kernel,
	})
	if err != nil {
		return "", fmt.Errorf("marshal WSL metadata: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
