// Package sample is a minimal example plugin for nanovms.
//
// Demonstrates the Plugin interface; used in tests and as a template
// for new plugins.
package sample

import (
	"context"
	"sync/atomic"

	"github.com/kooshapari/nanovms/internal/plugin"
)

// ID is the stable plugin identifier.
const ID plugin.ID = "phenotype.plugin.sample"

type sample struct {
	count atomic.Int64
}

func New() plugin.Plugin { return &sample{} }

func (s *sample) Info() plugin.Info {
	return plugin.Info{ID: ID, Name: "Sample Plugin", Version: "0.1.0"}
}

func (s *sample) Init(_ context.Context) error { return nil }

func (s *sample) Shutdown(_ context.Context) error { return nil }

func (s *sample) Health(_ context.Context) error { return nil }

// Tick increments the internal counter and returns the new value.
func (s *sample) Tick() int64 { return s.count.Add(1) }
