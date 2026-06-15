// SPDX-License-Identifier: MIT OR Apache-2.0
// Package validate provides configuration validation with schema checks for NVMS.
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kooshapari/nanovms/pkg/config"
)

// Validator performs schema and semantic validation on NVMS configurations.
type Validator struct {
	schema Schema
}

// Schema defines the validation rules.
type Schema struct {
	RequiredFields []string
	AllowedTiers   []int
	MinCPU         int
	MaxCPU         int
	MinMemory      int
	MaxMemory      int
	MinDisk        int
	MaxDisk        int
	AllowedImages  []string
}

// DefaultSchema returns the default NVMS validation schema.
func DefaultSchema() Schema {
	return Schema{
		RequiredFields: []string{"name", "image", "tier"},
		AllowedTiers:   []int{1, 2, 3},
		MinCPU:         2,
		MaxCPU:         64,
		MinMemory:      64,
		MaxMemory:      65536,
		MinDisk:        0,
		MaxDisk:        1048576,
	}
}

// NewValidator creates a new validator with the default schema.
func NewValidator() *Validator {
	return &Validator{schema: DefaultSchema()}
}

// NewValidatorWithSchema creates a validator with a custom schema.
func NewValidatorWithSchema(schema Schema) *Validator {
	return &Validator{schema: schema}
}

// Result holds the outcome of a validation run.
type Result struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// Validate performs full schema and semantic validation on a config.
func (v *Validator) Validate(cfg *config.NVMSConfig) *Result {
	result := &Result{Valid: true}

	// Required field checks
	for _, field := range v.schema.RequiredFields {
		switch field {
		case "name":
			if cfg.Name == "" {
				result.addError("name is required")
			}
		case "image":
			if cfg.Image == "" {
				result.addError("image is required")
			}
		case "tier":
			if cfg.Tier == 0 {
				result.addError("tier is required")
			}
		}
	}

	// Tier validation
	tierValid := false
	for _, t := range v.schema.AllowedTiers {
		if cfg.Tier == t {
			tierValid = true
			break
		}
	}
	if !tierValid {
		result.addError(fmt.Sprintf("tier %d is not allowed; must be one of %v", cfg.Tier, v.schema.AllowedTiers))
	}

	// Resource validation
	if cfg.CPU < v.schema.MinCPU {
		result.addError(fmt.Sprintf("cpu must be at least %d", v.schema.MinCPU))
	}
	if cfg.CPU > v.schema.MaxCPU {
		result.addError(fmt.Sprintf("cpu must be at most %d", v.schema.MaxCPU))
	}
	if cfg.Memory < v.schema.MinMemory {
		result.addError(fmt.Sprintf("memory must be at least %d MB", v.schema.MinMemory))
	}
	if cfg.Memory > v.schema.MaxMemory {
		result.addError(fmt.Sprintf("memory must be at most %d MB", v.schema.MaxMemory))
	}
	if cfg.Disk < v.schema.MinDisk {
		result.addError(fmt.Sprintf("disk must be at least %d MB", v.schema.MinDisk))
	}
	if cfg.Disk > v.schema.MaxDisk {
		result.addError(fmt.Sprintf("disk must be at most %d MB", v.schema.MaxDisk))
	}

	// Image validation
	if cfg.Image != "" {
		if err := validateImageRef(cfg.Image); err != nil {
			result.addError(fmt.Sprintf("invalid image reference: %v", err))
		}
		if len(v.schema.AllowedImages) > 0 {
			allowed := false
			for _, img := range v.schema.AllowedImages {
				if cfg.Image == img {
					allowed = true
					break
				}
			}
			if !allowed {
				result.addWarning(fmt.Sprintf("image %s is not in the allowed list", cfg.Image))
			}
		}
	}

	// Sandbox type validation
	if cfg.Sandbox.Type != "" {
		allowedTypes := []string{"vm", "container", "wasm", "process", "native"}
		found := false
		for _, t := range allowedTypes {
			if cfg.Sandbox.Type == t {
				found = true
				break
			}
		}
		if !found {
			result.addError(fmt.Sprintf("sandbox type %s is not valid", cfg.Sandbox.Type))
		}
	}

	// Mount validation
	for i, m := range cfg.Mounts {
		if m.Source == "" {
			result.addError(fmt.Sprintf("mounts[%d].source is required", i))
		}
		if m.Target == "" {
			result.addError(fmt.Sprintf("mounts[%d].target is required", i))
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateMinimal checks only required fields for quick validation.
func (v *Validator) ValidateMinimal(cfg *config.NVMSConfig) *Result {
	result := &Result{Valid: true}
	if cfg.Name == "" {
		result.addError("name is required")
	}
	if cfg.Image == "" {
		result.addError("image is required")
	}
	if cfg.Tier < 1 || cfg.Tier > 3 {
		result.addError("tier must be 1, 2, or 3")
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func (r *Result) addError(msg string) {
	r.Errors = append(r.Errors, msg)
}

func (r *Result) addWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// Error returns a single error string combining all validation errors.
func (r *Result) Error() string {
	if r.Valid {
		return ""
	}
	return fmt.Sprintf("validation failed with %d errors: %s", len(r.Errors), strings.Join(r.Errors, "; "))
}

// imageRefPattern matches a simple OCI-style image reference.
// Supports: name, name:tag, registry/name, registry/name:tag
var imageRefPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*(/[a-zA-Z0-9._-]+)*(:[a-zA-Z0-9._-]+)?$`)

func validateImageRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("image reference is empty")
	}
	if len(ref) > 255 {
		return fmt.Errorf("image reference exceeds 255 characters")
	}
	if !imageRefPattern.MatchString(ref) {
		return fmt.Errorf("image reference contains invalid characters")
	}
	return nil
}
