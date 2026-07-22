// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import (
	"context"

	"github.com/kooshapari/nanovms/pkg/gpu"
)

// Provider plans, applies, and verifies lockable NVMS environment profiles.
type Provider struct {
	Inventory gpu.InventoryProvider
	Toolkit   ToolkitResolver
	State     StateStore
}

// Plan inspects inventory, toolkit, and WSL metadata without mutating the host.
func (provider Provider) Plan(ctx context.Context, request Request) (Result, error) {
	result := newResult(request)
	contract, err := provider.buildContract(ctx, request)
	if err != nil {
		return failResult(result, err)
	}
	result.Success = true
	result.Contract = contract
	return result, nil
}

// Apply records planned mutations and is a no-op when the contract is already applied.
func (provider Provider) Apply(ctx context.Context, request Request) (Result, error) {
	result := newResult(request)
	contract, err := provider.buildContract(ctx, request)
	if err != nil {
		return failResult(result, err)
	}
	if provider.State != nil {
		if applied, ok := provider.State.AppliedDigest(request.Profile); ok && applied == contract.Digest {
			result.Success = true
			result.NoOp = true
			result.Contract = contract
			return result, nil
		}
		if err := provider.State.RecordApplied(request.Profile, contract.Digest); err != nil {
			return failResult(result, providerError(CodeApplyFailed, "record applied contract: %v", err))
		}
	}
	result.Success = true
	result.Contract = contract
	return result, nil
}

// Verify fails closed unless the planned contract matches the applied state.
func (provider Provider) Verify(ctx context.Context, request Request) (Result, error) {
	result := newResult(request)
	contract, err := provider.buildContract(ctx, request)
	if err != nil {
		return failResult(result, err)
	}
	if provider.State == nil {
		return failResult(result, providerError(CodeEnvironmentNotApplied, "no applied environment state for profile %q", request.Profile))
	}
	applied, ok := provider.State.AppliedDigest(request.Profile)
	if !ok || applied != contract.Digest {
		return failResult(result, providerError(CodeEnvironmentDrift, "applied environment does not match current plan for profile %q", request.Profile))
	}
	result.Success = true
	result.Contract = contract
	return result, nil
}

func (provider Provider) buildContract(ctx context.Context, request Request) (Contract, error) {
	if err := validateRequest(request); err != nil {
		return Contract{}, err
	}
	profile, err := ensureProfile(request.Profile)
	if err != nil {
		return Contract{}, err
	}
	if provider.Inventory == nil {
		return Contract{}, providerError(CodeInventoryFailed, "inventory provider is required")
	}
	devices, err := provider.Inventory.Inventory(ctx)
	if err != nil {
		return Contract{}, providerError(CodeInventoryFailed, "%v", err)
	}
	if len(devices) == 0 {
		return Contract{}, providerError(CodeInventoryEmpty, "GPU inventory returned no devices")
	}
	device, err := findDevice(devices, profile.GPUUUID)
	if err != nil {
		return Contract{}, err
	}
	toolkit, err := provider.Toolkit.Resolve(ctx, request, profile)
	if err != nil {
		return Contract{}, err
	}
	if err := validateCompatibility(device, profile, toolkit); err != nil {
		return Contract{}, err
	}
	var wsl *WSLMetadata
	if request.WSLDistribution != "" {
		wsl, err = InspectWSL(ctx, provider.Toolkit.Inspector.Runner, request.WSLDistribution)
		if err != nil {
			return Contract{}, err
		}
	}
	contract := Contract{
		Version:       ProviderVersion,
		Profile:       profile.ID,
		GPU:           device,
		Runtime:       runtimeIdentity(profile),
		Packages:      clonePackages(profile.Packages),
		Compatibility: compatibilityRecord(profile, device, toolkit, true),
		Mutations:     plannedMutations(toolkit),
		WSL:           wsl,
		Toolkit:       toolkitRecord(profile, toolkit),
	}
	digest, err := DigestContract(contract)
	if err != nil {
		return Contract{}, providerError(CodeContractDigestFailed, "%v", err)
	}
	contract.Digest = digest
	return contract, nil
}

// DefaultProvider wires the standard host toolkit inspector.
func DefaultProvider(inventory gpu.InventoryProvider, runner gpu.CommandRunner, state StateStore) Provider {
	return Provider{
		Inventory: inventory,
		Toolkit: ToolkitResolver{
			Inspector: HostToolkitInspector{Runner: runner},
		},
		State: state,
	}
}
