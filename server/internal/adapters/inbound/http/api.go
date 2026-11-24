package http

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

// Broadcaster delivers real-time notifications to connected clients.
type Broadcaster interface {
	BroadcastOperation(projectID string, edit timeline.Edit, state ports.State, author string)
	BroadcastAsset(a asset.Asset)
}

// Service defines the inbound port dependencies for HTTP endpoints.
type Service interface {
	ports.ProjectService
	ports.AssetManager
}

// API provides the HTTP transport layer for Kairo.
type API struct {
	service     Service
	exportSvc   ports.ExportService
	log         *slog.Logger
	broadcaster Broadcaster
}

// New registers API routes, applies middleware, and returns an nethttp.Handler.
func New(svc Service, exportSvc ports.ExportService, wsHandler nethttp.Handler, logger *slog.Logger) nethttp.Handler {
	var broadcaster Broadcaster
	if b, ok := wsHandler.(Broadcaster); ok {
		broadcaster = b
	}
	a := &API{service: svc, exportSvc: exportSvc, log: logger, broadcaster: broadcaster}
	mux := nethttp.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/health", a.handleHealth)

	// WebSocket real-time sync
	if wsHandler != nil {
		mux.Handle("GET /api/project/ws", wsHandler)
	}

	// Project & Operations
	mux.HandleFunc("GET /api/project", a.handleGetProject)
	mux.HandleFunc("POST /api/project/operations", a.handleCommitOperation)

	// Asset registration & lifecycle
	mux.HandleFunc("POST /api/assets", a.handleRegisterAsset)
	mux.HandleFunc("POST /api/assets/{id}/upload", a.handleBeginUpload)
	mux.HandleFunc("POST /api/assets/{id}/complete", a.handleCompleteUpload)
	mux.HandleFunc("POST /api/assets/{id}/fail", a.handleFailUpload)
	mux.HandleFunc("GET /api/assets/{id}/content", a.handleAssetContent)

	// Export lifecycle
	if exportSvc != nil {
		mux.HandleFunc("POST /api/exports", a.handleCreateExport)
		mux.HandleFunc("GET /api/exports/{id}", a.handleGetExport)
		mux.HandleFunc("POST /api/exports/{id}/cancel", a.handleCancelExport)
		mux.HandleFunc("POST /api/exports/{id}/retry", a.handleRetryExport)
		mux.HandleFunc("GET /api/exports/{id}/download", a.handleDownloadExport)
	}

	return a.withMiddleware(mux)
}

func (a *API) withMiddleware(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
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
