package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	domErrors "server/internal/domain/errors"
	"server/internal/service"
)

// API provides the HTTP transport layer for Kairo.
type API struct {
	service *service.Service
	log     *slog.Logger
}

// New registers API routes, applies middleware, and returns an http.Handler.
func New(svc *service.Service, logger *slog.Logger) http.Handler {
	a := &API{service: svc, log: logger}
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/health", a.handleHealth)

	// Project & Operations
	mux.HandleFunc("GET /api/project", a.handleGetProject)
	mux.HandleFunc("POST /api/project/operations", a.handleCommitOperation)

	// Asset registration & lifecycle
	mux.HandleFunc("POST /api/assets", a.handleRegisterAsset)
	mux.HandleFunc("POST /api/assets/{id}/upload", a.handleBeginUpload)
	mux.HandleFunc("POST /api/assets/{id}/complete", a.handleCompleteUpload)
	mux.HandleFunc("POST /api/assets/{id}/fail", a.handleFailUpload)
	mux.HandleFunc("GET /api/assets/{id}/content", a.handleAssetContent)

	return a.withMiddleware(mux)
}

func (a *API) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == "POST" && r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			a.reply(w, r, 0, nil, domErrors.ErrInvalid)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
