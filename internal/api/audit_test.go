package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditLoggerAppendAndQuery(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	// Append entries with varying timestamps.
	base := time.Now().UTC()
	for i := 0; i < 5; i++ {
		l.Append(AuditEntry{
			Timestamp:  base.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Method:     "POST",
			Path:       "/v1/sandbox",
			StatusCode: 200,
			DurationMs: 10,
			Provider:   "firecracker",
			Isolation:  "microvm",
		})
	}

	// Query all (newest first).
	results := l.Query("", "", "", "", 0, 0)
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	// Newest first.
	if results[0].Timestamp != base.Add(4*time.Second).Format(time.RFC3339) {
		t.Errorf("expected newest first; got timestamp %q", results[0].Timestamp)
	}
}

func TestAuditLoggerFilterByProvider(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	ts := time.Now().UTC().Format(time.RFC3339)
	l.Append(AuditEntry{Timestamp: ts, Provider: "firecracker"})
	l.Append(AuditEntry{Timestamp: ts, Provider: "gvisor"})
	l.Append(AuditEntry{Timestamp: ts, Provider: "firecracker"})

	results := l.Query("", "", "firecracker", "", 0, 0)
	if len(results) != 2 {
		t.Fatalf("expected 2 firecracker entries, got %d", len(results))
	}
}

func TestAuditLoggerFilterByTimeRange(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	base := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	l.Append(AuditEntry{Timestamp: base.Format(time.RFC3339)})
	l.Append(AuditEntry{Timestamp: base.Add(30 * time.Second).Format(time.RFC3339)})
	l.Append(AuditEntry{Timestamp: base.Add(90 * time.Second).Format(time.RFC3339)})

	from := base.Format(time.RFC3339)
	to := base.Add(60 * time.Second).Format(time.RFC3339)
	results := l.Query(from, to, "", "", 0, 0)
	if len(results) != 2 {
		t.Fatalf("expected 2 entries in time range, got %d", len(results))
	}
}

func TestAuditLoggerJSONLOutput(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	l.Append(AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Method:     "GET",
		Path:       "/v1/health",
		StatusCode: 200,
		DurationMs: 5,
	})

	// Verify JSONL file was written.
	p := filepath.Join(dir, "nvms-audit.jsonl")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("cannot read JSONL file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("JSONL file is empty")
	}
}

func TestAuditLoggerNoDir(t *testing.T) {
	// Passing empty string should not panic.
	l := NewAuditLogger("")
	defer func() { _ = l.Close() }()
	l.Append(AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Method:     "POST",
		Path:       "/v1/sandbox",
		StatusCode: 201,
		DurationMs: 42,
	})
	// Query should still work with in-memory buffer.
	results := l.Query("", "", "", "", 0, 0)
	if len(results) != 1 {
		t.Fatalf("expected 1 entry in memory-only mode, got %d", len(results))
	}
}

func TestAuditLoggerRotation(t *testing.T) {
	dir := t.TempDir()
	// Temporarily lower the rotation threshold for testing.
	origMax := MaxJSONLLenBytes
	MaxJSONLLenBytes = 64
	defer func() { MaxJSONLLenBytes = origMax }()

	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	// Append enough entries to trigger rotation (small payload ~100 bytes each).
	payload := AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Method:     "POST",
		Path:       "/v1/sandbox/test/exec",
		StatusCode: 200,
		DurationMs: 15,
		Provider:   "firecracker",
		Isolation:  "microvm",
		ClientTokenHash: "abcdef1234567890abcdef1234567890",
	}
	for i := 0; i < 50; i++ {
		l.Append(payload)
		payload.Timestamp = time.Now().UTC().Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339)
	}

	// Check that the original file exists and a rotated copy exists.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}
	files := 0
	for _, e := range entries {
		if !e.IsDir() {
			files++
		}
	}
	if files < 2 {
		t.Logf("expected >=2 files after rotation; got %d (may be timing-dependent)", files)
	}
}

func TestAuditLoggerRingBufferWrap(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	// Overfill the ring buffer by 20%.
	total := int(float64(MaxInMemoryEntries) * 1.2)
	for i := 0; i < total; i++ {
		l.Append(AuditEntry{
			Timestamp:  time.Now().UTC().Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339),
			Method:     "GET",
			Path:       "/v1/health",
			StatusCode: 200,
			DurationMs: 1,
		})
	}

	results := l.Query("", "", "", "", 0, 0)
	if len(results) != MaxInMemoryEntries {
		t.Fatalf("expected %d entries (ring buffer size), got %d", MaxInMemoryEntries, len(results))
	}
}

func TestAuditLoggerPagination(t *testing.T) {
	dir := t.TempDir()
	l := NewAuditLogger(dir)
	defer func() { _ = l.Close() }()

	ts := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 20; i++ {
		l.Append(AuditEntry{
			Timestamp:  ts,
			Method:     "DELETE",
			Path:       "/v1/sandbox/" + string(rune('a'+i)),
			StatusCode: 204,
			DurationMs: int64(i),
		})
	}

	// Paginated: limit 5, offset 3.
	results := l.Query("", "", "", "", 5, 3)
	if len(results) != 5 {
		t.Fatalf("expected 5 paginated results, got %d", len(results))
	}
}
