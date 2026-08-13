package podman

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	got := parseTime("2026-07-30T04:00:00.123456Z")
	if got.IsZero() || !got.Equal(time.Date(2026, 7, 30, 4, 0, 0, 123456000, time.UTC)) {
		t.Fatalf("unexpected timestamp: %v", got)
	}
	if got := parseTime("0001-01-01T00:00:00Z"); !got.IsZero() {
		t.Fatalf("zero timestamp parsed as %v", got)
	}
}

func TestParseBytes(t *testing.T) {
	tests := map[string]int64{
		"12 B":    12,
		"1.5 MiB": 1572864,
		"2 GB":    2147483648,
		"unknown": 0,
		"":        0,
	}
	for input, want := range tests {
		if got := parseBytes(input); got != want {
			t.Errorf("parseBytes(%q)=%d, want %d", input, got, want)
		}
	}
}

func TestNewAdapterWithBinary(t *testing.T) {
	if got := NewAdapterWithBinary("/usr/bin/podman").binary; got != "/usr/bin/podman" {
		t.Fatalf("binary=%q", got)
	}
}
