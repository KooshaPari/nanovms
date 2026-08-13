package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEntry records a single NVMS service request for observability.
type AuditEntry struct {
	ID              int64  `json:"id"`
	Timestamp       string `json:"timestamp"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	RequestID       string `json:"request_id,omitempty"`
	StatusCode      int    `json:"status_code"`
	DurationMs      int64  `json:"duration_ms"`
	ClientTokenHash string `json:"client_token_hash,omitempty"`
	Provider        string `json:"provider,omitempty"`
	Isolation       string `json:"isolation,omitempty"`
	Error           string `json:"error,omitempty"`
}

func newAuditRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(value[:])
}

const (
	// MaxInMemoryEntries controls how many entries the ring buffer retains.
	MaxInMemoryEntries = 10000
)

var (
	// MaxJSONLLenBytes triggers rotation at this JSONL file size.
	// Exported as var to allow test overrides.
	MaxJSONLLenBytes int64 = 100 * 1024 * 1024 // 100 MB
)

// AuditLogger maintains an in-memory ring buffer of audit entries and
// writes them to a rotating JSONL file.
type AuditLogger struct {
	mu        sync.RWMutex
	buf       []AuditEntry
	head      int
	count     int
	jsonlPath string
	jsonlFile *os.File
}

// NewAuditLogger creates an AuditLogger that writes JSONL to the given
// data directory (does not rotate by default; call maybeRotate after
// each write).
func NewAuditLogger(dataDir string) *AuditLogger {
	l := &AuditLogger{
		buf:   make([]AuditEntry, MaxInMemoryEntries),
		head:  0,
		count: 0,
	}
	if dataDir != "" {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("[audit] warning: cannot create %s: %v", dataDir, err)
			return l
		}
		p := filepath.Join(dataDir, "nvms-audit.jsonl")
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("[audit] warning: cannot open %s: %v", p, err)
			return l
		}
		l.jsonlPath = p
		l.jsonlFile = f
	}
	return l
}

// Append records a new entry in the ring buffer and writes it to JSONL.
func (l *AuditLogger) Append(e AuditEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf[l.head] = e
	l.head = (l.head + 1) % MaxInMemoryEntries
	if l.count < MaxInMemoryEntries {
		l.count++
	}

	if l.jsonlFile != nil {
		line, err := json.Marshal(e)
		if err == nil {
			_, _ = l.jsonlFile.Write(append(line, '\n'))
			l.maybeRotate()
		}
	}
}

// Query returns audit entries matching the given filters, in reverse
// chronological order (newest first), paginated.
func (l *AuditLogger) Query(from, to string, provider, isolation string, limit, offset int) []AuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Build index range, newest first.
	var filtered []AuditEntry
	start := (l.head - 1 + MaxInMemoryEntries) % MaxInMemoryEntries
	for i := 0; i < l.count; i++ {
		idx := (start - i + MaxInMemoryEntries) % MaxInMemoryEntries
		e := l.buf[idx]

		if from != "" && e.Timestamp < from {
			break // entries are monotonic; no need to go further
		}
		if to != "" && e.Timestamp > to {
			continue
		}
		if provider != "" && e.Provider != provider {
			continue
		}
		if isolation != "" && e.Isolation != isolation {
			continue
		}
		filtered = append(filtered, e)
	}

	// Paginate.
	if offset > len(filtered) {
		return nil
	}
	filtered = filtered[offset:]
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered
}

// Close flushes and closes the JSONL writer.
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.jsonlFile != nil {
		return l.jsonlFile.Close()
	}
	return nil
}

func (l *AuditLogger) maybeRotate() {
	if l.jsonlFile == nil {
		return
	}
	info, err := l.jsonlFile.Stat()
	if err != nil {
		return
	}
	if info.Size() < MaxJSONLLenBytes {
		return
	}

	// Close current, rename, open new.
	_ = l.jsonlFile.Close()
	ts := time.Now().UTC().Format("20060102T150405Z")
	rotated := fmt.Sprintf("%s.%s", l.jsonlPath, ts)
	if err := os.Rename(l.jsonlPath, rotated); err != nil {
		log.Printf("[audit] rotation rename failed: %v", err)
	}
	f, err := os.OpenFile(l.jsonlPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[audit] rotation reopen failed: %v", err)
		return
	}
	l.jsonlFile = f
}
