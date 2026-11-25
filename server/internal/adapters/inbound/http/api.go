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
	BroadcastProjectAsset(projectID string, a asset.Asset)
	BroadcastProjectAssetDeleted(projectID, id string)
}

// WSHandler combines WebSocket serving and room broadcasting.
type WSHandler interface {
	Broadcaster
	ServeWs(w nethttp.ResponseWriter, r *nethttp.Request)
}

// API provides the HTTP transport layer for Kairo.
type API struct {
	manager     ports.ProjectManager
	exportSvc   ports.ExportService
	log         *slog.Logger
	broadcaster Broadcaster
}

// New registers API routes, applies middleware, and returns an nethttp.Handler.
func New(manager ports.ProjectManager, exportSvc ports.ExportService, wsHandler WSHandler, logger *slog.Logger) nethttp.Handler {
	var broadcaster Broadcaster
	if wsHandler != nil {
		broadcaster = wsHandler
	}
	a := &API{manager: manager, exportSvc: exportSvc, log: logger, broadcaster: broadcaster}
	mux := nethttp.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/health", a.handleHealth)

	// Isolated Rooms & Project-scoped Routes
	mux.HandleFunc("POST /api/rooms", a.handleCreateRoom)
	mux.HandleFunc("POST /api/projects", a.handleCreateProject)
	mux.HandleFunc("GET /api/projects/{projectId}", a.handleGetProjectByID)
	mux.HandleFunc("DELETE /api/projects/{projectId}", a.handleDeleteProjectByID)

	mux.HandleFunc("POST /api/projects/{projectId}/operations", a.handleCommitOperationByProject)
	mux.HandleFunc("POST /api/projects/{projectId}/assets", a.handleRegisterAssetByProject)
	mux.HandleFunc("POST /api/projects/{projectId}/assets/{id}/upload", a.handleBeginUploadByProject)
	mux.HandleFunc("POST /api/projects/{projectId}/assets/{id}/complete", a.handleCompleteUploadByProject)
	mux.HandleFunc("POST /api/projects/{projectId}/assets/{id}/fail", a.handleFailUploadByProject)
	mux.HandleFunc("GET /api/projects/{projectId}/assets/{id}/content", a.handleAssetContentByProject)

	// WebSocket real-time sync (strictly project-scoped, calls ServeWs directly)
	if wsHandler != nil {
		mux.HandleFunc("GET /api/projects/{projectId}/ws", wsHandler.ServeWs)
	}

	// Export lifecycle
	if exportSvc != nil {
		mux.HandleFunc("POST /api/projects/{projectId}/exports", a.handleCreateExportByProject)
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
