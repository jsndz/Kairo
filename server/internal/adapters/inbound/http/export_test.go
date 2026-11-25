package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "server/internal/adapters/inbound/http"
	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/ports"
)

type mockExportService struct {
	submitFunc func(ctx context.Context, req ports.SubmitRequest) (domainExport.Job, error)
	getFunc    func(ctx context.Context, id string) (domainExport.Job, error)
	cancelFunc func(ctx context.Context, id string) (domainExport.Job, error)
	retryFunc  func(ctx context.Context, id string) (domainExport.Job, error)
	dlFunc     func(ctx context.Context, id string) (string, error)
}

func (m *mockExportService) Submit(ctx context.Context, req ports.SubmitRequest) (domainExport.Job, error) {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, req)
	}
	return domainExport.Job{ID: "job-123", RequestID: req.RequestID, Status: domainExport.JobStatusQueued}, nil
}

func (m *mockExportService) Get(ctx context.Context, id string) (domainExport.Job, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return domainExport.Job{ID: id, Status: domainExport.JobStatusRendering, Progress: 45.0}, nil
}

func (m *mockExportService) Cancel(ctx context.Context, id string) (domainExport.Job, error) {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, id)
	}
	return domainExport.Job{ID: id, Status: domainExport.JobStatusCancelled}, nil
}

func (m *mockExportService) Retry(ctx context.Context, id string) (domainExport.Job, error) {
	if m.retryFunc != nil {
		return m.retryFunc(ctx, id)
	}
	return domainExport.Job{ID: id, Status: domainExport.JobStatusQueued, Attempts: 2}, nil
}

func (m *mockExportService) DownloadURL(ctx context.Context, id string) (string, error) {
	if m.dlFunc != nil {
		return m.dlFunc(ctx, id)
	}
	return "https://minio.test/download/video.mp4", nil
}

func TestExportHTTPRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mockExp := &mockExportService{}
	handler := httpadapter.New(nil, mockExp, nil, logger)

	t.Run("POST /api/projects/{projectId}/exports valid request", func(t *testing.T) {
		payload := map[string]any{
			"requestId":  "req-1",
			"revision":   5,
			"filename":   "final.mp4",
			"resolution": "1080p",
			"frameRate":  30,
			"quality":    "standard",
			"method":     "server",
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/exports", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /api/projects/{projectId}/exports stale revision 409 conflict", func(t *testing.T) {
		mockExp.submitFunc = func(_ context.Context, _ ports.SubmitRequest) (domainExport.Job, error) {
			return domainExport.Job{}, domErrors.ErrConflict
		}
		defer func() { mockExp.submitFunc = nil }()

		payload := map[string]any{
			"requestId": "req-stale",
			"revision":  2,
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/exports", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /api/exports/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/exports/job-123", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var job domainExport.Job
		if err := json.NewDecoder(rec.Body).Decode(&job); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if job.ID != "job-123" || job.Status != domainExport.JobStatusRendering {
			t.Fatalf("unexpected job response: %+v", job)
		}
	})

	t.Run("POST /api/exports/{id}/cancel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/exports/job-123/cancel", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var job domainExport.Job
		_ = json.NewDecoder(rec.Body).Decode(&job)
		if job.Status != domainExport.JobStatusCancelled {
			t.Fatalf("expected cancelled, got %s", job.Status)
		}
	})

	t.Run("POST /api/exports/{id}/retry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/exports/job-123/retry", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var job domainExport.Job
		_ = json.NewDecoder(rec.Body).Decode(&job)
		if job.Attempts != 2 {
			t.Fatalf("expected 2 attempts, got %d", job.Attempts)
		}
	})

	t.Run("GET /api/exports/{id}/download redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/exports/job-123/download", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected 307 redirect, got %d", rec.Code)
		}
		if rec.Header().Get("Location") != "https://minio.test/download/video.mp4" {
			t.Fatalf("unexpected redirect: %s", rec.Header().Get("Location"))
		}
	})
}
