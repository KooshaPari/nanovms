// SPDX-License-Identifier: MIT OR Apache-2.0
// Package containers implements lifecycle adapters for native OCI-compatible
// container CLIs that are not Docker-daemon APIs.
package containers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/ports"
)

// Kind identifies a native container CLI command dialect.
type Kind string

const (
	KindApple Kind = "apple-containers"
	KindWSL   Kind = "wsl-containers"
)

// Runner is the narrow process boundary used by Adapter. Tests can inject a
// deterministic implementation without spawning a shell or a host runtime.
type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
	Stream(context.Context, string, ...string) (io.ReadCloser, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, args...).CombinedOutput()
}

func (execRunner) Stream(ctx context.Context, binary string, args ...string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stderr = cmd.Stdout
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &waitReadCloser{ReadCloser: pipe, wait: cmd.Wait}, nil
}

// Adapter implements ports.SandboxPort using an explicit native CLI.
type Adapter struct {
	kind   Kind
	binary string
	runner Runner
	lookup func(string) error
}

var _ ports.SandboxPort = (*Adapter)(nil)

// NewAppleAdapter creates an adapter for Apple's container CLI.
func NewAppleAdapter(binary string) *Adapter {
	if strings.TrimSpace(binary) == "" {
		binary = "container"
	}
	return newAdapter(KindApple, binary, execRunner{}, func(path string) error {
		_, err := exec.LookPath(path)
		return err
	})
}

// NewWSLAdapter creates an adapter for Microsoft's first-party WSL container
// CLI. NVMS_WSLC_BINARY can override the executable for packaged installs.
func NewWSLAdapter(binary string) *Adapter {
	if strings.TrimSpace(binary) == "" {
		binary = os.Getenv("NVMS_WSLC_BINARY")
	}
	if strings.TrimSpace(binary) == "" {
		binary = "wslc.exe"
	}
	return newAdapter(KindWSL, binary, execRunner{}, func(path string) error {
		_, err := exec.LookPath(path)
		return err
	})
}

// NewAdapterWithRunner creates a deterministic adapter for tests and
// controlled embedding. The injected runner is never invoked through a shell.
func NewAdapterWithRunner(kind Kind, binary string, runner Runner) *Adapter {
	if runner == nil {
		runner = execRunner{}
	}
	return newAdapter(kind, binary, runner, func(string) error { return nil })
}

func newAdapter(kind Kind, binary string, runner Runner, lookup func(string) error) *Adapter {
	return &Adapter{kind: kind, binary: binary, runner: runner, lookup: lookup}
}

// Probe verifies the binary and its native version command without creating a
// container.
func (a *Adapter) Probe(ctx context.Context) error {
	if a == nil || a.runner == nil || a.lookup == nil {
		return fmt.Errorf("container adapter is not configured")
	}
	if err := a.lookup(a.binary); err != nil {
		return fmt.Errorf("%s: binary %q not found: %w", a.kind, a.binary, err)
	}
	args := []string{"version"}
	if a.kind == KindApple {
		args = []string{"system", "version", "--format", "json"}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := a.runner.Run(probeCtx, a.binary, args...); err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%s: probe timed out", a.kind)
		}
		return fmt.Errorf("%s: probe failed: %w", a.kind, err)
	}
	return nil
}

// Create uses each native CLI's stopped-create operation. The API layer owns
// the subsequent Start call, so a failed start can be rolled back safely.
func (a *Adapter) Create(ctx context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if err := a.Probe(ctx); err != nil {
		return nil, err
	}
	args, err := a.createArgs(cfg)
	if err != nil {
		return nil, err
	}
	out, err := a.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		id = cfg.Name
	}
	return a.Get(ctx, id)
}

// Start starts a stopped container and is idempotent for a running container.
func (a *Adapter) Start(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s start: id is required", a.kind)
	}
	sb, err := a.Get(ctx, id)
	if err != nil {
		return err
	}
	if sb.Status == domain.SandboxStatusRunning {
		return nil
	}
	_, err = a.run(ctx, a.command("start", id)...)
	return err
}

// Stop stops a container, using kill for a forced stop.
func (a *Adapter) Stop(ctx context.Context, id string, force bool) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s stop: id is required", a.kind)
	}
	verb := "stop"
	if force {
		verb = "kill"
	}
	_, err := a.run(ctx, a.command(verb, id)...)
	return err
}

// Delete removes a container and its metadata.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s delete: id is required", a.kind)
	}
	return a.runDiscard(ctx, a.command("delete", "--force", id)...)
}

// List returns all native containers, including stopped containers.
func (a *Adapter) List(ctx context.Context) ([]*domain.Sandbox, error) {
	args := a.listArgs()
	out, err := a.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	ids := listIDs(out)
	result := make([]*domain.Sandbox, 0, len(ids))
	for _, id := range ids {
		sb, getErr := a.Get(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		result = append(result, sb)
	}
	return result, nil
}

// Get returns native inspect state for id.
func (a *Adapter) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%s inspect: id is required", a.kind)
	}
	out, err := a.run(ctx, a.command("inspect", id)...)
	if err != nil {
		return nil, err
	}
	return decodeSandbox(out)
}

// Logs streams native container logs.
func (a *Adapter) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	args := a.command("logs")
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, id)
	return a.runner.Stream(ctx, a.binary, args...)
}

// Exec executes command inside a running container.
func (a *Adapter) Exec(ctx context.Context, id string, command []string) (io.ReadCloser, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("%s exec: command is required", a.kind)
	}
	args := append(a.command("exec", id), command...)
	return a.runner.Stream(ctx, a.binary, args...)
}

// Metrics returns a best-effort one-shot stats sample.
func (a *Adapter) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	out, err := a.run(ctx, a.command("stats", "--format", "json", "--no-stream", id)...)
	if err != nil {
		return nil, err
	}
	metrics, err := decodeMetrics(out, id)
	if err != nil {
		return nil, fmt.Errorf("%s stats: %w", a.kind, err)
	}
	return metrics, nil
}

func (a *Adapter) run(ctx context.Context, args ...string) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := a.runner.Run(runCtx, a.binary, args...)
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%s %s timed out", a.kind, strings.Join(args, " "))
		}
		return nil, fmt.Errorf("%s %s: %w: %s", a.kind, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func (a *Adapter) runDiscard(ctx context.Context, args ...string) error {
	_, err := a.run(ctx, args...)
	return err
}

func (a *Adapter) command(operation string, ids ...string) []string {
	prefix := []string{operation}
	if a.kind == KindWSL {
		prefix = []string{"container", operation}
		if operation == "delete" {
			prefix[1] = "remove"
		}
	}
	return append(prefix, ids...)
}

func (a *Adapter) listArgs() []string {
	if a.kind == KindWSL {
		return []string{"container", "list", "--all", "--quiet"}
	}
	return []string{"list", "--all", "--quiet"}
}

func (a *Adapter) createArgs(cfg domain.SandboxConfig) ([]string, error) {
	if a.kind == KindWSL && cfg.ReadOnlyRootfs {
		return nil, fmt.Errorf("wsl-containers: read-only rootfs is unsupported by the installed CLI")
	}
	args := append(a.command("create"), "--name", cfg.Name)
	for _, key := range sortedKeys(cfg.Environment) {
		args = append(args, "--env", key+"="+cfg.Environment[key])
	}
	for _, key := range sortedKeys(cfg.Labels) {
		args = append(args, "--label", key+"="+cfg.Labels[key])
	}
	for _, mount := range cfg.Mounts {
		if mount.Source == "" || mount.Target == "" {
			continue
		}
		spec := "type=" + defaultMountType(mount.Type) + ",source=" + mount.Source + ",target=" + mount.Target
		if mount.ReadOnly {
			spec += ",readonly"
		}
		args = append(args, "--mount", spec)
	}
	if cfg.WorkDir != "" {
		args = append(args, "--workdir", cfg.WorkDir)
	}
	if cfg.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}
	if cfg.TmpfsTmp {
		args = append(args, "--tmpfs", "/tmp")
	}
	args = append(args, cfg.Image)
	if cfg.NativeSandbox != nil {
		args = append(args, cfg.NativeSandbox.Command...)
	}
	return args, nil
}

func validateConfig(cfg domain.SandboxConfig) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("native container: name is required")
	}
	if strings.TrimSpace(cfg.Image) == "" {
		return fmt.Errorf("native container: image is required")
	}
	for _, mount := range cfg.Mounts {
		if strings.TrimSpace(mount.Source) == "" || strings.TrimSpace(mount.Target) == "" {
			return fmt.Errorf("native container: mount source and target are required")
		}
	}
	return nil
}

func defaultMountType(value string) string {
	if strings.TrimSpace(value) == "" {
		return "bind"
	}
	return value
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func listIDs(out []byte) []string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	var values []any
	if json.Unmarshal([]byte(text), &values) == nil {
		result := make([]string, 0, len(values))
		for _, value := range values {
			if object, ok := value.(map[string]any); ok {
				if id := fieldString(object, "id", "Id", "ID"); id != "" {
					result = append(result, id)
				}
			} else if id, ok := value.(string); ok && id != "" {
				result = append(result, id)
			}
		}
		return result
	}
	result := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		if id := strings.TrimSpace(line); id != "" {
			result = append(result, id)
		}
	}
	return result
}

func decodeSandbox(out []byte) (*domain.Sandbox, error) {
	var raw any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &raw); err != nil {
		return nil, fmt.Errorf("native container inspect: decode: %w", err)
	}
	if values, ok := raw.([]any); ok {
		if len(values) == 0 {
			return nil, fmt.Errorf("native container inspect: empty result")
		}
		raw = values[0]
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("native container inspect: expected object")
	}
	id := fieldString(object, "id", "Id", "ID")
	name := strings.TrimPrefix(fieldString(object, "name", "Name"), "/")
	if id == "" {
		id = name
	}
	if name == "" {
		name = id
	}
	status := strings.ToLower(fieldString(object, "status", "Status"))
	if stateObject := fieldObject(object, "state", "State"); stateObject != nil {
		status = strings.ToLower(fieldString(stateObject, "status", "Status"))
	}
	state := domain.SandboxStatusStopped
	switch status {
	case "running":
		state = domain.SandboxStatusRunning
	case "created", "configured", "initialized", "pending":
		state = domain.SandboxStatusPending
	case "stopped", "exited", "dead", "removing":
		state = domain.SandboxStatusStopped
	}
	configuration := fieldObject(object, "configuration", "Configuration")
	labels := fieldMap(object, "labels", "Labels")
	if len(labels) == 0 {
		labels = fieldMap(configuration, "labels", "Labels")
	}
	if config := fieldObject(object, "config", "Config"); config != nil {
		if len(labels) == 0 {
			labels = fieldMap(config, "labels", "Labels")
		}
	}
	image := fieldString(object, "image", "Image")
	if imageObject := fieldObject(configuration, "image", "Image"); imageObject != nil {
		image = fieldString(imageObject, "reference", "Reference", "name", "Name")
	}
	if config := fieldObject(object, "config", "Config"); config != nil && image == "" {
		image = fieldString(config, "image", "Image")
	}
	created := parseTime(fieldString(object, "created", "Created"))
	started := parseTime(fieldString(object, "started_at", "StartedAt", "startedAt"))
	statusObject := fieldObject(object, "status", "Status")
	if statusObject != nil {
		if statusValue := fieldString(statusObject, "state", "State"); statusValue != "" {
			status = strings.ToLower(statusValue)
			switch status {
			case "running":
				state = domain.SandboxStatusRunning
			case "created", "configured", "initialized", "pending":
				state = domain.SandboxStatusPending
			default:
				state = domain.SandboxStatusStopped
			}
		}
		if started.IsZero() {
			started = parseTime(fieldString(statusObject, "startedDate", "started_at", "StartedAt"))
		}
	}
	ipAddress := inspectIPAddress(object, statusObject, configuration)
	var startedAt *time.Time
	if !started.IsZero() {
		startedAt = &started
	}
	return &domain.Sandbox{
		ID: id, Name: name, Status: state, Type: domain.SandboxTypeContainer,
		Config:    &domain.SandboxConfig{Name: name, Image: image, Labels: labels},
		CreatedAt: created, StartedAt: startedAt,
		IPAddress: ipAddress,
	}, nil
}

func decodeMetrics(out []byte, id string) (*domain.SandboxMetrics, error) {
	var raw any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &raw); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if values, ok := raw.([]any); ok {
		if len(values) == 0 {
			return nil, fmt.Errorf("empty result")
		}
		raw = values[0]
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object")
	}
	memoryText := fieldString(object, "MemUsage", "memory_usage", "memory")
	memory := parseBytes(memoryText)
	if memory == 0 {
		memory = integer(object, "memory", "memory_usage", "mem_usage")
	}
	return &domain.SandboxMetrics{
		SandboxID:   id,
		CPUUsage:    number(object, "cpu", "cpu_percent", "CPU %", "CPUPerc"),
		MemoryUsage: memory,
		NetworkRx:   parseBytes(fieldString(object, "network_rx", "NetInput", "rx_bytes")),
		NetworkTx:   parseBytes(fieldString(object, "network_tx", "NetOutput", "tx_bytes")),
	}, nil
}

func inspectIPAddress(object, status, configuration map[string]any) string {
	if value := fieldString(object, "ip_address", "IPAddress", "ip"); value != "" {
		return value
	}
	for _, candidate := range []map[string]any{status, fieldObject(object, "network_settings", "NetworkSettings")} {
		if value := fieldString(candidate, "ip_address", "IPAddress", "ipv4Address", "IPv4Address"); value != "" {
			return value
		}
		for _, network := range fieldArray(candidate, "networks", "Networks") {
			if entry, ok := network.(map[string]any); ok {
				if value := fieldString(entry, "ip_address", "IPAddress", "ipv4Address", "IPv4Address"); value != "" {
					return value
				}
			}
		}
	}
	for _, network := range fieldArray(configuration, "networks", "Networks") {
		if entry, ok := network.(map[string]any); ok {
			if value := fieldString(entry, "ip_address", "IPAddress", "ipv4Address", "IPv4Address"); value != "" {
				return value
			}
		}
	}
	return ""
}

func fieldObject(object map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		for actual, value := range object {
			if strings.EqualFold(actual, key) {
				if nested, ok := value.(map[string]any); ok {
					return nested
				}
			}
		}
	}
	return nil
}

func fieldArray(object map[string]any, keys ...string) []any {
	for _, key := range keys {
		for actual, value := range object {
			if strings.EqualFold(actual, key) {
				if nested, ok := value.([]any); ok {
					return nested
				}
			}
		}
	}
	return nil
}

func fieldString(object map[string]any, keys ...string) string {
	for _, key := range keys {
		for actual, value := range object {
			if strings.EqualFold(actual, key) {
				if text, ok := value.(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func fieldMap(object map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		for actual, value := range object {
			if !strings.EqualFold(actual, key) {
				continue
			}
			raw, ok := value.(map[string]any)
			if !ok {
				continue
			}
			result := make(map[string]string, len(raw))
			for name, entry := range raw {
				if text, ok := entry.(string); ok {
					result[name] = text
				}
			}
			return result
		}
	}
	return nil
}

func number(values map[string]any, keys ...string) float64 {
	for _, key := range keys {
		for actual, value := range values {
			if strings.EqualFold(actual, key) {
				switch typed := value.(type) {
				case float64:
					return typed
				case string:
					var parsed float64
					_, _ = fmt.Sscanf(strings.TrimSuffix(typed, "%"), "%f", &parsed)
					return parsed
				}
			}
		}
	}
	return 0
}

func integer(values map[string]any, keys ...string) int64 {
	for _, key := range keys {
		for actual, value := range values {
			if strings.EqualFold(actual, key) {
				switch typed := value.(type) {
				case float64:
					return int64(typed)
				case string:
					var parsed int64
					_, _ = fmt.Sscanf(typed, "%d", &parsed)
					return parsed
				}
			}
		}
	}
	return 0
}

func parseBytes(value string) int64 {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	// Native CLIs commonly emit "1.2MiB / 2GiB"; only the used value is
	// meaningful for the one-shot sandbox sample.
	text = strings.Fields(strings.SplitN(text, "/", 2)[0])[0]
	if text == "" {
		return 0
	}
	upper := strings.ToUpper(strings.TrimSpace(text))
	unit := ""
	for _, candidate := range []string{"KIB", "MIB", "GIB", "TIB", "KB", "MB", "GB", "TB", "B"} {
		if strings.HasSuffix(upper, candidate) {
			unit = candidate
			upper = strings.TrimSpace(strings.TrimSuffix(upper, candidate))
			break
		}
	}
	valueNumber, err := strconv.ParseFloat(upper, 64)
	if err != nil {
		return 0
	}
	multiplier := float64(1)
	switch unit {
	case "KIB":
		multiplier = 1 << 10
	case "MIB":
		multiplier = 1 << 20
	case "GIB":
		multiplier = 1 << 30
	case "TIB":
		multiplier = 1 << 40
	case "KB":
		multiplier = 1e3
	case "MB":
		multiplier = 1e6
	case "GB":
		multiplier = 1e9
	case "TB":
		multiplier = 1e12
	}
	return int64(valueNumber * multiplier)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05 -0700 MST"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

type waitReadCloser struct {
	io.ReadCloser
	wait func() error
}

func (r *waitReadCloser) Close() error {
	closeErr := r.ReadCloser.Close()
	waitErr := r.wait()
	if closeErr != nil {
		return closeErr
	}
	return waitErr
}
