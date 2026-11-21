package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/httpapi"
	"server/internal/service"
	"server/internal/storage/catalog"
	"server/internal/storage/jsonl"
)

type mockObjects struct {
	data map[string]asset.ObjectInfo
}

func (m *mockObjects) Bucket() string { return "test-bucket" }
func (m *mockObjects) UploadPolicy(_ context.Context, key string, _ asset.Source) (asset.Upload, error) {
	return asset.Upload{URL: "https://minio.test/upload", Fields: map[string]string{"key": key}}, nil
}
func (m *mockObjects) Stat(_ context.Context, key string) (asset.ObjectInfo, error) {
	if info, ok := m.data[key]; ok {
		return info, nil
	}
	return asset.ObjectInfo{}, domErrors.ErrNotFound
}
func (m *mockObjects) Promote(_ context.Context, stage, key, etag string) error {
	if m.data[stage].ETag != etag {
		return domErrors.ErrConflict
	}
	m.data[key] = m.data[stage]
	return nil
}
func (m *mockObjects) ReadURL(_ context.Context, _ string) (string, error) {
	return "https://minio.test/read", nil
}

func setupAPI(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	l := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}
	obj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	svc, err := service.Open(cat, l, obj, 10<<20)
	if err != nil {
		t.Fatalf("failed to open service: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httpapi.New(svc, logger)
}

func TestHealthCheck(t *testing.T) {
	handler := setupAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %q", body["status"])
	}
}

func TestGetProject(t *testing.T) {
	handler := setupAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var state service.State
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode project state: %v", err)
	}
	if state.ID == "" {
		t.Fatalf("expected non-empty project ID")
	}
}

func TestAssetRegistrationAndValidation(t *testing.T) {
	handler := setupAPI(t)

	validSource := asset.Source{
		Name:     "clip.mp4",
		Kind:     "video",
		MIME:     "video/mp4",
		Bytes:    5000,
		Frames:   300,
		Duration: 10,
	}
	bodyBytes, _ := json.Marshal(validSource)
	req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var a asset.Asset
	if err := json.NewDecoder(rec.Body).Decode(&a); err != nil {
		t.Fatalf("failed to decode asset: %v", err)
	}
	if a.ID == "" || a.Status != "pending" {
		t.Fatalf("unexpected asset state: %+v", a)
	}

	// Invalid content type
	reqBadCT := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(bodyBytes))
	reqBadCT.Header.Set("Content-Type", "text/plain")
	recBadCT := httptest.NewRecorder()
	handler.ServeHTTP(recBadCT, reqBadCT)
	if recBadCT.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for bad Content-Type, got %d", recBadCT.Code)
	}

	// Cross-site fetch protection
	reqCSRF := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(bodyBytes))
	reqCSRF.Header.Set("Content-Type", "application/json")
	reqCSRF.Header.Set("Sec-Fetch-Site", "cross-site")
	recCSRF := httptest.NewRecorder()
	handler.ServeHTTP(recCSRF, reqCSRF)
	if recCSRF.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for cross-site post, got %d", recCSRF.Code)
	}
}
