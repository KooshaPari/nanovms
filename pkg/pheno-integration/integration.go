// SPDX-License-Identifier: MIT OR Apache-2.0
//
// Package phenointegration is a thin HTTP middleware helper used by the
// nanovms daemon and any future pheno-internal services to attach a
// per-request X-Request-Id header (echoing inbound, generating otherwise)
// and structured logging via the standard `log/slog` package.
//
// Historically this package depended on `github.com/kooshapari/pheno-go-ctxkit/ctxkit`,
// but that dependency pulled a non-trivial graph (and required a
// `replace` directive to a worktree path that only existed on the original
// author's machine). The ctxkit pattern is small enough to inline
// directly here, which makes `nanovms` self-contained and reproducible to
// build from a clean clone. The package remains import-compatible: any
// caller of `InitServer(ctx)` continues to get the same http.Handler
// shape.
package phenointegration

import (
	"context"
	"log/slog"
	"net/http"
)

// requestIDHeader is the HTTP header used to carry the per-request
// identifier end-to-end.
const requestIDHeader = "X-Request-Id"

// InitServer returns an http.Handler with the request-id middleware
// applied. The returned handler registers /healthz and can be passed
// directly to http.Server or wrapped by additional middleware.
//
// The middleware is intentionally tiny: it generates a UUID-like ID per
// request, threads it through the request context (so downstream
// handlers can read it via RequestIDFrom), echoes it on the response,
// and emits a single structured log line at request completion.
func InitServer(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HandleHealthz)
	return requestIDMiddleware(mux)
}

// HandleHealthz is a simple liveness probe that returns HTTP 200.
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// contextKey is unexported to prevent key collisions in the request
// context.
type contextKey string

const requestIDKey contextKey = "requestID"

// RequestIDFrom returns the per-request identifier previously attached by
// requestIDMiddleware. Returns the empty string if no ID is present.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// requestIDMiddleware wraps an http.Handler so every request gets a
// stable X-Request-Id header. Inbound requests that already carry the
// header have it preserved; otherwise a 16-byte hex ID is generated.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		slog.Info("http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", id),
		)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
