//go:build gofuzz
// +build gofuzz

// Package fuzztest provides go-fuzz targets for the sandbox adapter.
//
// Run with: go-fuzz-build && go-fuzz -bin=./sandbox-fuzz -workdir=./fuzzworkdir
package fuzztest

import (
    "github.com/kooshapari/nanovms/internal/adapters/sandbox"
    "github.com/kooshapari/nanovms/internal/domain"
)

// FuzzAdapterCreate fuzzes the Adapter.Create path.
func FuzzAdapterCreate(data []byte) int {
    a := sandbox.NewAdapter()
    if a == nil {
        return 0
    }
    cfg := domain.SandboxConfig{
        Name:   string(data),
        Image:  "alpine:latest",
        Status: domain.SandboxTypeVM,
    }
    sb, err := a.Create(nil, cfg)
    if err != nil {
        return 0 // invalid input is fine
    }
    if sb == nil || sb.ID == "" {
        return 1 // should always have an ID
    }
    return 0
}
