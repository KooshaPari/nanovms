package plugin_test

import (
	"context"
	"testing"

	"github.com/kooshapari/nanovms/internal/plugin"
	"github.com/kooshapari/nanovms/internal/plugin/sample"
)

func TestRegistry_RegisterAndFind(t *testing.T) {
	r := plugin.NewRegistry()
	p := sample.New()
	if err := r.Register(context.Background(), p); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if r.Len() != 1 {
		t.Fatalf("Len = %d, want 1", r.Len())
	}
	got, ok := r.Find(sample.ID)
	if !ok {
		t.Fatalf("Find: not found")
	}
	if got.Info().ID != sample.ID {
		t.Fatalf("ID = %s, want %s", got.Info().ID, sample.ID)
	}
}

func TestRegistry_UnregisterAll(t *testing.T) {
	r := plugin.NewRegistry()
	_ = r.Register(context.Background(), sample.New())
	_ = r.Register(context.Background(), sample.New())
	if r.Len() != 2 {
		t.Fatalf("Len before = %d, want 2", r.Len())
	}
	r.UnregisterAll(context.Background())
	if r.Len() != 0 {
		t.Fatalf("Len after = %d, want 0", r.Len())
	}
}

func TestRegistry_FindMissing(t *testing.T) {
	r := plugin.NewRegistry()
	_, ok := r.Find("nonexistent")
	if ok {
		t.Fatalf("Find nonexistent: ok = true, want false")
	}
}
