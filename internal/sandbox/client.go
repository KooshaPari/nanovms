// Package sandbox provides a UDS client for the NVMS daemon.
//
// The client is used by the nvms CLI (cmd/nvms) to talk to a running
// daemon over its Unix domain socket. It mirrors the daemon's HTTP
// surface defined in internal/api/router.go.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// Client is a UDS HTTP client for the NVMS daemon.
type Client struct {
	socketPath string
	http       http.Client
}

// NewClient returns a Client that dials the daemon's UDS.
// If socketPath is empty, it defaults to $XDG_RUNTIME_DIR/nanovms/routed.sock.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		runDir := os.Getenv("XDG_RUNTIME_DIR")
		if runDir == "" {
			runDir = "/tmp"
		}
		socketPath = runDir + "/nanovms/routed.sock"
	}
	return &Client{
		socketPath: socketPath,
		http: http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

// ListSandboxes returns all sandboxes from the daemon.
func (c *Client) ListSandboxes(ctx context.Context) ([]SandboxInfo, error) {
	resp, err := c.http.Get("http://nvms/v1/sandboxes")
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		Object string        `json:"object"`
		Data   []SandboxInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return body.Data, nil
}

// Exec executes a command in a sandbox and returns a ReadCloser of output.
func (c *Client) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	body, _ := json.Marshal(map[string]any{"command": cmd})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://nvms/v1/sandboxes/"+id+"/exec", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exec %s: %w", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("exec %s: HTTP %d", id, resp.StatusCode)
	}
	return resp.Body, nil
}

// Logs streams logs from a sandbox.
func (c *Client) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	u := fmt.Sprintf("http://nvms/v1/sandboxes/%s/logs?follow=%t", id, follow)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("logs %s: %w", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("logs %s: HTTP %d", id, resp.StatusCode)
	}
	return resp.Body, nil
}

// PortForward creates a port-forward tunnel to a sandbox.
// Returns the local proxy address.
func (c *Client) PortForward(ctx context.Context, id string, localPort, remotePort int) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"local_port":  localPort,
		"remote_port": remotePort,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://nvms/v1/sandboxes/"+id+"/port-forward", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("port-forward %s: %w", id, err)
	}
	defer resp.Body.Close()

	var result struct {
		LocalAddress string `json:"local_address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.LocalAddress, nil
}

// SandboxInfo is the JSON shape returned by the daemon for sandbox listing.
type SandboxInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type,omitempty"`
}
