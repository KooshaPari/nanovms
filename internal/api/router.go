// Package api wires the NVMS daemon's HTTP surface.
//
// It depends only on the hexagonal ports (SandboxPort) and the token.Manager
// for auth. All handlers are wired into a single http.Handler returned by
// NewRouter. The daemon is consumed over a Unix domain socket by the
// byteport-engine NVMS adapter (T2 UDS RPC binding tier).
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/listen"
	"github.com/kooshapari/nanovms/internal/ports"
	"github.com/kooshapari/nanovms/internal/token"
)

// Handlers bundles the port implementations and auth manager.
type Handlers struct {
	Port  ports.SandboxPort
	Token *token.Manager
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
	mux.HandleFunc("/v1/models", auth(h.Token, h.handleModels))
	mux.HandleFunc("/v1/proxy/dispatch", auth(h.Token, h.handleProxyDispatch))
	mux.HandleFunc("/v1/deploy", auth(h.Token, h.handleDeploy))
	mux.HandleFunc("/v1/stop", auth(h.Token, h.handleStop))
	mux.HandleFunc("/v1/sandboxes", auth(h.Token, h.handleListSandboxes))
	mux.HandleFunc("/v1/metrics", auth(h.Token, h.handleMetrics))

	// ID-scoped sandbox routes.
	mux.HandleFunc("/v1/sandboxes/", auth(h.Token, h.handleSandboxByID))

	return mux
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
	id := r.URL.Path[len("/v1/sandboxes/"):]
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
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

func (h Handlers) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# HELP nvms_up NVMS daemon is up\nnvms_up 1\n"))
}

// auth wraps a handler with bearer-token auth.
func auth(tm *token.Manager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if tm == nil {
			// Unconfigured: deny by default.
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}
		authz := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(authz) <= len(prefix) || authz[:len(prefix)] != prefix {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := tm.Check(authz[len(prefix):]); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
