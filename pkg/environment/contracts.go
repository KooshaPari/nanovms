// SPDX-License-Identifier: MIT OR Apache-2.0
// Package environment is the authoritative NVMS WSL/CUDA environment provider.
package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

const ProviderVersion = "nanovms.io/environment/v1"

// ProfileID names one lockable environment profile.
type ProfileID string

const (
	ProfileAmpere ProfileID = "ampere"
	ProfilePascal ProfileID = "pascal"
)

// Request is the machine-readable environment provider boundary.
type Request struct {
	Version         string            `json:"version"`
	Profile         ProfileID         `json:"profile"`
	WSLDistribution string            `json:"wsl_distribution,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	ToolkitRoot     string            `json:"toolkit_root,omitempty"`
}

// RuntimeIdentity records interpreter/runtime placeholders for the profile.
type RuntimeIdentity struct {
	PythonInterpreter string `json:"python_interpreter,omitempty"`
	TorchVariant      string `json:"torch_variant,omitempty"`
}

// PackageDigest records a package identity digest when available.
type PackageDigest struct {
	Name   string `json:"name"`
	Digest string `json:"digest,omitempty"`
}

// CompatibilityRecord captures driver and toolkit compatibility evidence.
type CompatibilityRecord struct {
	CUDAToolkitRequested string `json:"cuda_toolkit_requested"`
	CUDAToolkitObserved  string `json:"cuda_toolkit_observed,omitempty"`
	DriverVersion        string `json:"driver_version,omitempty"`
	DriverCUDACeiling    string `json:"driver_cuda_ceiling,omitempty"`
	ComputeCapability    string `json:"compute_capability,omitempty"`
	Compatible           bool   `json:"compatible"`
}

// Mutation records one planned or applied environment change.
type Mutation struct {
	Kind  string `json:"kind"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// ToolkitRecord captures requested and observed CUDA toolkit evidence.
type ToolkitRecord struct {
	Requested  string `json:"requested"`
	Observed   string `json:"observed,omitempty"`
	Root       string `json:"root,omitempty"`
	Executable string `json:"executable,omitempty"`
}

// WSLMetadata records read-only WSL distribution and kernel facts.
type WSLMetadata struct {
	Distribution string `json:"distribution"`
	Kernel       string `json:"kernel,omitempty"`
	Digest       string `json:"digest"`
}

// Contract is the canonical environment state for one profile.
type Contract struct {
	Version       string              `json:"version"`
	Profile       ProfileID           `json:"profile"`
	GPU           gpu.Device          `json:"gpu"`
	Runtime       RuntimeIdentity     `json:"runtime"`
	Packages      []PackageDigest     `json:"packages,omitempty"`
	Compatibility CompatibilityRecord `json:"compatibility"`
	Mutations     []Mutation          `json:"mutations,omitempty"`
	WSL           *WSLMetadata        `json:"wsl,omitempty"`
	Toolkit       ToolkitRecord       `json:"toolkit"`
	Digest        string              `json:"digest"`
}

// Result is always safe to serialize as machine-readable evidence.
type Result struct {
	Version      string   `json:"version"`
	Success      bool     `json:"success"`
	NoOp         bool     `json:"no_op,omitempty"`
	Contract     Contract `json:"contract,omitempty"`
	ErrorCode    string   `json:"error_code,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// ProviderError carries a stable machine-readable failure code.
type ProviderError struct {
	Code string
	Err  error
}

func (e *ProviderError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *ProviderError) Unwrap() error { return e.Err }

func providerError(code, format string, values ...any) error {
	return &ProviderError{Code: code, Err: fmt.Errorf(format, values...)}
}

func newResult(request Request) Result {
	return Result{Version: ProviderVersion}
}

func failResult(result Result, err error) (Result, error) {
	var providerErr *ProviderError
	code := "environment_failed"
	message := err.Error()
	if errors.As(err, &providerErr) {
		code = providerErr.Code
		message = providerErr.Err.Error()
	}
	result.Success = false
	result.ErrorCode = code
	result.ErrorMessage = message
	return result, err
}

type canonicalPackage struct {
	Name string `json:"name"`
}

type canonicalContract struct {
	Version       string                `json:"version"`
	Profile       ProfileID             `json:"profile"`
	GPU           gpu.Device            `json:"gpu"`
	Runtime       RuntimeIdentity       `json:"runtime"`
	Packages      []canonicalPackage    `json:"packages,omitempty"`
	Compatibility CompatibilityRecord   `json:"compatibility"`
	Mutations     []Mutation            `json:"mutations,omitempty"`
	WSL           *canonicalWSLMetadata `json:"wsl,omitempty"`
	Toolkit       ToolkitRecord         `json:"toolkit"`
}

// CanonicalJSON validates and deterministically serializes the contract body
// excluding the digest field.
func (contract Contract) CanonicalJSON() ([]byte, error) {
	if contract.Version != ProviderVersion {
		return nil, fmt.Errorf("unsupported environment contract version %q", contract.Version)
	}
	if err := contract.GPU.UUID.Validate(); err != nil {
		return nil, err
	}
	packages := make([]canonicalPackage, len(contract.Packages))
	for i, pkg := range contract.Packages {
		packages[i] = canonicalPackage{Name: pkg.Name}
	}
	var wsl *canonicalWSLMetadata
	if contract.WSL != nil {
		wsl = &canonicalWSLMetadata{
			Distribution: contract.WSL.Distribution,
			Kernel:       contract.WSL.Kernel,
		}
	}
	body := canonicalContract{
		Version:       contract.Version,
		Profile:       contract.Profile,
		GPU:           contract.GPU,
		Runtime:       contract.Runtime,
		Packages:      packages,
		Compatibility: contract.Compatibility,
		Mutations:     append([]Mutation(nil), contract.Mutations...),
		WSL:           wsl,
		Toolkit:       contract.Toolkit,
	}
	sort.Slice(body.Packages, func(i, j int) bool { return body.Packages[i].Name < body.Packages[j].Name })
	sort.Slice(body.Mutations, func(i, j int) bool {
		left, right := body.Mutations[i], body.Mutations[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Key != right.Key {
			return left.Key < right.Key
		}
		return left.Value < right.Value
	})
	sort.Slice(body.GPU.Observations, func(i, j int) bool {
		left, right := body.GPU.Observations[i], body.GPU.Observations[j]
		if left.Scope == right.Scope {
			if left.ScopeID == right.ScopeID {
				return left.Index < right.Index
			}
			return left.ScopeID < right.ScopeID
		}
		return left.Scope < right.Scope
	})
	return json.Marshal(body)
}

// DigestContract computes the SHA-256 digest of the canonical contract JSON.
func DigestContract(contract Contract) (string, error) {
	payload, err := contract.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
