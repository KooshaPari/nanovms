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
	token      string
	http       http.Client
}

// NewClient returns a Client that dials the daemon's UDS.
// If socketPath is empty, it defaults to $XDG_RUNTIME_DIR/nanovms/routed.sock.
func NewClient(socketPath string) *Client {
	return newClient(socketPath, "")
}

// NewClientWithToken returns a Client that authenticates requests with a
// static daemon bearer token. An empty token preserves the unauthenticated
// behavior of NewClient for health or test-only callers.
func NewClientWithToken(socketPath, token string) *Client {
	return newClient(socketPath, token)
}

func newClient(socketPath, token string) *Client {
	if socketPath == "" {
		runDir := os.Getenv("XDG_RUNTIME_DIR")
		if runDir == "" {
			runDir = "/tmp"
		}
		socketPath = runDir + "/nanovms/routed.sock"
	}
	return &Client{
		socketPath: socketPath,
		token:      token,
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

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://nvms"+path, body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// ListSandboxes returns all sandboxes from the daemon.
func (c *Client) ListSandboxes(ctx context.Context) ([]SandboxInfo, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/sandboxes", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list sandboxes: HTTP %d", resp.StatusCode)
	}

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
	body, err := json.Marshal(map[string]any{"command": cmd})
	if err != nil {
		return nil, fmt.Errorf("encode exec %s: %w", id, err)
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/sandboxes/"+id+"/exec", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exec %s: %w", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("exec %s: HTTP %d", id, resp.StatusCode)
	}
	return resp.Body, nil
}

// Logs streams logs from a sandbox.
func (c *Client) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/logs?follow=%t", id, follow)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("logs %s: %w", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
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
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/sandboxes/"+id+"/port-forward", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("port-forward %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

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
