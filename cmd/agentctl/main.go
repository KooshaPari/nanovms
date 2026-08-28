// SPDX-License-Identifier: MIT OR Apache-2.0
// agentctl: a single-binary JSON-in / JSON-out CLI that exposes the nanovms
// daemon's port-trait operations. Wire-compatible with the Omniroute
// dispatcher schema: {"method": "...", "params": {...}} in -> {"ok": bool, "result": ...} out.
//
// Usage:
//   echo '{"method":"sandbox.create","params":{"name":"hello"}}' | nanovms-agentctl
//
// Supported methods (initial cut):
//   - sandbox.create   -> domain.SandboxConfig -> SandboxPort.Create
//   - sandbox.list     -> ([]string)            -> SandboxPort.List
//   - sandbox.get      -> SandboxID            -> SandboxPort.Get
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type request struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type response struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Err    string      `json:"error,omitempty"`
}

// handler is a function that takes a request and returns a result or error.
type handler func(ctx context.Context, r *request) (interface{}, error)

// handlers is the dispatch table; placeholder handlers for the MVP.
var handlers = map[string]handler{
	"sandbox.create": func(ctx context.Context, r *request) (interface{}, error) {
		name, _ := r.Params["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("sandbox.create: missing 'name' param")
		}
		// Stub: real implementation would call daemon.SandboxPort.Create
		return map[string]interface{}{
			"id":        fmt.Sprintf("sb-%d", time.Now().UnixNano()),
			"name":      name,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		}, nil
	},
	"sandbox.list": func(ctx context.Context, r *request) (interface{}, error) {
		// Stub: real implementation would call daemon.SandboxPort.List
		return []map[string]interface{}{}, nil
	},
	"sandbox.get": func(ctx context.Context, r *request) (interface{}, error) {
		id, _ := r.Params["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("sandbox.get: missing 'id' param")
		}
		return map[string]interface{}{"id": id, "status": "running"}, nil
	},
}

func dispatch(r *request) response {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	h, ok := handlers[r.Method]
	if !ok {
		return response{OK: false, Err: fmt.Sprintf("unknown method: %s", r.Method)}
	}
	res, err := h(ctx, r)
	if err != nil {
		return response{OK: false, Err: err.Error()}
	}
	return response{OK: true, Result: res}
}

func main() {
	in := os.Stdin
	out := io.Writer(os.Stdout)
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		_, _ = fmt.Fprintln(out, "agentctl: JSON-in/JSON-out Omniroute-compatible dispatcher. Read stdin, write stdout.")
		os.Exit(0)
	}
	scanner := bufio.NewScanner(in)
	// Allow long lines (e.g., large manifests)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		line := scanner.Bytes()
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(response{OK: false, Err: "invalid json: " + err.Error()})
			continue
		}
		_ = enc.Encode(dispatch(&req))
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "agentctl: read error:", err)
		os.Exit(1)
	}
}
