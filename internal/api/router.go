// Package api wires the NVMS daemon's HTTP surface.
//
// It depends only on the hexagonal ports (SandboxPort) and the token.Manager
// for auth. All handlers are wired into a single http.Handler returned by
// NewRouter. The daemon is consumed over a Unix domain socket by the
// byteport-engine NVMS adapter (T2 UDS RPC binding tier).
//
// Phase 2 additions:
//   - AuditLogger middleware (records every request)
//   - Rate-limit middleware (100 req/s token bucket)
//   - GET /v1/audit endpoint (filtered audit-log query)
//   - 429 Too Many Requests on rate-limit exceed
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/ports"
	"github.com/kooshapari/nanovms/internal/token"
)

// Handlers bundles the port implementations and auth manager.
type Handlers struct {
	Port        ports.SandboxPort
	Token       *token.Manager
	JWTVerifier *token.JWTVerifier
	AuditLog    *AuditLogger
}

// NewRouter builds the daemon's HTTP handler.
//
// Endpoints (11):
//
//	GET  /healthz                  -> liveness (no auth)
//	GET  /readyz                   -> readiness (no auth)
//	GET  /v1/models                -> model catalog (auth)
//	POST /v1/proxy/dispatch        -> proxy dispatch (auth)
//	POST /v1/deploy                -> deploy sandbox (auth)
//	POST /v1/stop                  -> stop sandbox (auth)
//	GET  /v1/sandboxes             -> list sandboxes (auth)
//	POST /v1/sandboxes             -> create sandbox (auth)
//	GET  /v1/sandboxes/{id}        -> get sandbox (auth)
//	DELETE /v1/sandboxes/{id}      -> delete sandbox (auth)
//	GET  /v1/metrics               -> prometheus text (auth)
func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/readyz", h.handleReadyz)
	mux.HandleFunc("/v1/models", auth(h.Token, h.JWTVerifier, h.handleModels))
	mux.HandleFunc("/v1/proxy/dispatch", auth(h.Token, h.JWTVerifier, h.handleProxyDispatch))
	mux.HandleFunc("/v1/deploy", auth(h.Token, h.JWTVerifier, h.handleDeploy))
	mux.HandleFunc("/v1/stop", auth(h.Token, h.JWTVerifier, h.handleStop))
	mux.HandleFunc("/v1/sandboxes", auth(h.Token, h.JWTVerifier, h.handleListSandboxes))
	mux.HandleFunc("/v1/metrics", auth(h.Token, h.JWTVerifier, h.handleMetrics))

	// ID-scoped sandbox routes (exact match before wildcard).
	mux.HandleFunc("/v1/sandboxes/", auth(h.Token, h.JWTVerifier, h.handleSandboxByID))

	// Sub-routes on sandboxes: /v1/sandboxes/{id}/exec, /logs, /port-forward
	// These are handled within handleSandboxByID by checking path suffix.
	// handlePortForward delegates to the adapter's PortForward method when available.

	// Phase 2: audit-log query endpoint (auth-protected).
	if h.AuditLog != nil {
		mux.HandleFunc("/v1/audit", auth(h.Token, h.JWTVerifier, h.handleAudit))
	}

	// ── middleware chain ────────────────────────────────────────────
	var hdl http.Handler = mux

	// Audit-log: wraps every request to record it.
	if h.AuditLog != nil {
		hdl = auditMiddleware(h.AuditLog, hdl)
	}

	// Rate-limit: outermost (reject before any work).
	hdl = RateLimitMiddleware(hdl)

	return hdl
}

// Serve starts the HTTP server on the provided UDS listener.
func Serve(ctx context.Context, ln *listen.Listener, h Handlers) error {
	srv := &http.Server{Handler: NewRouter(h)}
	return ln.Serve(srv)
}

// ---- Handlers ----

func (h Handlers) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h Handlers) handleReadyz(w http.ResponseWriter, r *http.Request) {
	// For readiness, ping the port (if it implements a health method).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (h Handlers) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Phase 1b stub: returns a minimal model catalog. Real catalog comes
	// from the port's model registry once wired.
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   []map[string]string{},
	})
}

func (h Handlers) handleProxyDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Phase 1b stub: accept and acknowledge. Real dispatch fans out to the
	// configured provider behind the port.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "dispatch queued",
	})
}

func (h Handlers) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg domain.SandboxConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sb, err := h.Port.Create(r.Context(), cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// The deploy contract is synchronous: a successful response must describe
	// a sandbox that has actually been handed to the adapter's lifecycle.  A
	// Create-only response would leave callers with a misleading pending
	// sandbox, so always invoke Start before acknowledging the deployment.
	if err := h.Port.Start(r.Context(), sb.ID); err != nil {
		// Preserve the failed object in the response so callers can distinguish
		// an adapter capability/startup failure from a request or auth failure.
		sb.Status = domain.SandboxStatusFailed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   err.Error(),
			"sandbox": sb,
			"status":  domain.SandboxStatusFailed,
		})
		return
	}
	// Adapters normally update their stored object from Start. Normalize the
	// returned descriptor as well: a successful Start is the lifecycle
	// boundary, even for adapters whose Create state is named "created" rather
	// than "pending".
	if sb.Status != domain.SandboxStatusFailed {
		sb.Status = domain.SandboxStatusRunning
		if sb.StartedAt == nil {
			now := time.Now().UTC()
			sb.StartedAt = &now
		}
	}
	if sb.Status != domain.SandboxStatusRunning {
		http.Error(w, "adapter reported a non-running sandbox after start", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sb)
}

func (h Handlers) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if err := h.Port.Stop(r.Context(), id, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped", "id": id})
}

func (h Handlers) handleListSandboxes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := h.Port.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": list})
}

func (h Handlers) handleSandboxByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/v1/sandboxes/"):]
	id, sub, _ := strings.Cut(path, "/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// Sub-resource dispatch (exec / logs / port-forward).
	if sub != "" {
		switch sub {
		case "exec":
			h.handleExec(w, r, id)
			return
		case "logs":
			h.handleLogs(w, r, id)
			return
		case "port-forward":
			h.handlePortForward(w, r, id)
			return
		default:
			http.Error(w, "unknown sub-resource", http.StatusNotFound)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		sb, err := h.Port.Get(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sb)
	case http.MethodDelete:
		if err := h.Port.Delete(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Handlers) handleExec(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Command []string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	out, err := h.Port.Exec(r.Context(), id, req.Command)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, out)
	_ = out.Close()
}

func (h Handlers) handleLogs(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	follow := r.URL.Query().Get("follow") == "true"
	out, err := h.Port.Logs(r.Context(), id, follow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, out)
	_ = out.Close()
}

// portForwarder is implemented by adapters that support port-forwarding.
type portForwarder interface {
	PortForward(ctx context.Context, id string, localPort, remotePort int) (string, error)
}

func (h Handlers) handlePortForward(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		LocalPort  int `json:"local_port"`
		RemotePort int `json:"remote_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// Delegate to the adapter if it supports port-forwarding.
	if pf, ok := h.Port.(portForwarder); ok {
		addr, err := pf.PortForward(r.Context(), id, req.LocalPort, req.RemotePort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"local_address": addr})
		return
	}
	// Fallback: adapter does not support port-forwarding.
	http.Error(w, "port-forward not supported by this adapter", http.StatusNotImplemented)
}

func (h Handlers) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# HELP nvms_up NVMS daemon is up\nnvms_up 1\n"))
}

// handleAudit returns filtered audit-log entries.
func (h Handlers) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.AuditLog == nil {
		http.Error(w, "audit log not configured", http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = v
	}

	entries := h.AuditLog.Query(
		q.Get("from"), q.Get("to"),
		q.Get("provider"), q.Get("isolation"),
		limit, offset,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object":      "list",
		"object_type": "audit_log",
		"total":       len(entries),
		"limit":       limit,
		"offset":      offset,
		"entries":     entries,
	})
}

// auditMiddleware records every request through the audit logger.
func auditMiddleware(al *AuditLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		al.Append(AuditEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: 0,
			DurationMs: time.Since(start).Milliseconds(),
		})
	})
}

// auth wraps a handler with bearer-token auth, preferring JWT when configured.
func auth(tm *token.Manager, jv *token.JWTVerifier, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(authz) <= len(prefix) || authz[:len(prefix)] != prefix {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := authz[len(prefix):]

		// JWT verifier takes priority when configured.
		if jv != nil {
			claims, err := jv.Verify(tokenStr)
			if err != nil {
				http.Error(w, "forbidden: "+err.Error(), http.StatusForbidden)
				return
			}
			// Inject JWT claims into context for downstream handlers.
			ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
			next(w, r.WithContext(ctx))
			return
		}

		// Fallback: static token manager.
		if tm == nil {
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}
		if err := tm.Check(tokenStr); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

type ctxKey string

const ctxKeyClaims ctxKey = "jwt_claims"

// ClaimsFromContext extracts JWT claims from a request context, returning nil if absent.
func ClaimsFromContext(ctx context.Context) *token.JWTClaims {
	v, _ := ctx.Value(ctxKeyClaims).(*token.JWTClaims)
	return v
}
