// SPDX-License-Identifier: MIT OR Apache-2.0
// Package domain benchmarks: micro-benchmarks for the most-trafficked value types.
//
// Run with: go test -bench=. ./internal/domain/

package domain

import (
	"testing"
)

// BenchmarkSandboxConfig_New measures creation + population of SandboxConfig.
func BenchmarkSandboxConfig_New(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SandboxConfig{
			Name:        "bench-sandbox",
			Image:       "alpine:latest",
			VMType:      VMFlavorLima,
			SandboxType: SandboxTypeVM,
			WorkDir:     "/work",
		}
	}
}

// BenchmarkSandboxConfig_Clone measures deep-copy via assignment (struct is value-type).
func BenchmarkSandboxConfig_Clone(b *testing.B) {
	src := SandboxConfig{
		Name: "src", Image: "alpine:latest", VMType: VMFlavorLima, SandboxType: SandboxTypeVM,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = src // value semantics = copy
	}
}

// BenchmarkPortMapping_Slice measures slice-of-struct population.
func BenchmarkPortMapping_Slice(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ports := make([]PortMapping, 0, 8)
		for j := 0; j < 8; j++ {
			ports = append(ports, PortMapping{HostPort: j + 1000, ContainerPort: j + 8000, Protocol: "tcp"})
		}
		_ = ports
	}
}

// BenchmarkSandboxID_Len exercises the typical ID-format check path.
func BenchmarkSandboxID_Len(b *testing.B) {
	id := "sb-1234567890abcdef"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(id) >= 10
	}
}
