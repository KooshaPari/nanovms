// SPDX-License-Identifier: MIT OR Apache-2.0
// Package phenointegration wires the canonical Pheno Go context kit
// (github.com/kooshapari/pheno-go-ctxkit) into nanovms so every HTTP
// handler automatically receives request-scoped identifiers and logging.
package phenointegration

import (
	"context"
	"net/http"

	"github.com/kooshapari/pheno-go-ctxkit/ctxkit"
)

// InitServer returns an http.Handler with the ctxkit request-id middleware
// applied. The returned handler registers /healthz and can be passed
// directly to http.Server or wrapped by additional middleware.
func InitServer(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", HandleHealthz)
	return ctxkit.Middleware(mux)
}

// HandleHealthz is a simple liveness probe that returns HTTP 200.
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
