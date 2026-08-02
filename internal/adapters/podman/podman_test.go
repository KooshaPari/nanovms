package podman

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRunEnforcesCommandDeadline(t *testing.T) {
	if os.Getenv("NANOVMS_PODMAN_TIMEOUT_HELPER") == "1" {
		time.Sleep(time.Second)
		return
	}
	previous := podmanCommandTimeout
	podmanCommandTimeout = 20 * time.Millisecond
	t.Cleanup(func() { podmanCommandTimeout = previous })
	if err := os.Setenv("NANOVMS_PODMAN_TIMEOUT_HELPER", "1"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("NANOVMS_PODMAN_TIMEOUT_HELPER") })

	adapter := NewAdapterWithBinary(os.Args[0])
	_, err := adapter.run(context.Background(), "-test.run=^TestRunEnforcesCommandDeadline$", "-test.v")
	if err == nil {
		t.Fatal("expected bounded Podman command to time out")
	}
	if got := err.Error(); got != "podman -test.run=^TestRunEnforcesCommandDeadline$ timed out after 20ms" {
		t.Fatalf("unexpected timeout error: %s", got)
	}
}

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
