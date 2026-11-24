package http_test

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

	httpadapter "server/internal/adapters/inbound/http"
	"server/internal/adapters/outbound/storage/catalog"
	"server/internal/adapters/outbound/storage/jsonl"
	"server/internal/app/project"
	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/ports"
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
func (m *mockObjects) Download(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockObjects) Upload(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func setupAPI(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	l := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}
	obj := &mockObjects{data: make(map[string]asset.ObjectInfo)}

	svc, err := project.Open(cat, l, obj, 10<<20)
	if err != nil {
		t.Fatalf("failed to open service: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httpadapter.New(svc, nil, nil, logger)
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

	var state ports.State
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode project state: %v", err)
	}
	if state.ID == "" {
		t.Fatalf("expected non-empty project ID")
	}
}

func TestAssetRegistrationAndValidation(t *testing.T) {
	handler := setupAPI(t)

	// Valid video registration
	validBody := map[string]any{
		"name":     "clip.mp4",
		"kind":     "video",
		"mime":     "video/mp4",
		"bytes":    1024 * 1024,
		"frames":   300,
		"duration": 10.0,
		"width":    1920,
		"height":   1080,
	}
	bodyBytes, _ := json.Marshal(validBody)
	req := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var res asset.Asset
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode asset response: %v", err)
	}
	if res.ID == "" || res.Status != "pending" {
		t.Fatalf("unexpected asset response: %+v", res)
	}

	// Invalid media (wrong mime for kind)
	invalidBody := map[string]any{
		"name":     "bad.mp4",
		"kind":     "video",
		"mime":     "image/png",
		"bytes":    1024,
		"frames":   30,
		"duration": 1.0,
	}
	badBytes, _ := json.Marshal(invalidBody)
	reqBad := httptest.NewRequest(http.MethodPost, "/api/assets", bytes.NewReader(badBytes))
	reqBad.Header.Set("Content-Type", "application/json")
	recBad := httptest.NewRecorder()

	handler.ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid mime, got %d", recBad.Code)
	}

	// Content-Type enforcement
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

type inMemoryMockService struct {
	state ports.State
}

func (m *inMemoryMockService) Get() ports.State {
	return m.state
}

func (m *inMemoryMockService) GetOpsSince(_ uint64) ([]record.Record, error) {
	return nil, nil
}

func (m *inMemoryMockService) ProjectID() string {
	return m.state.ID
}

func (m *inMemoryMockService) Commit(_ timeline.Edit) (ports.State, error) {
	m.state.Revision++
	return m.state, nil
}

func (m *inMemoryMockService) CommitAuthor(_ timeline.Edit, _ string, _ ports.OwnershipChecker) (ports.State, error) {
	m.state.Revision++
	return m.state, nil
}

func (m *inMemoryMockService) Register(src asset.Source) (asset.Asset, error) {
	return asset.Asset{ID: "mock-asset-1", Source: src, Status: "pending"}, nil
}

func (m *inMemoryMockService) Begin(_ context.Context, id string) (asset.Asset, asset.Upload, error) {
	return asset.Asset{ID: id, Status: "uploading"}, asset.Upload{URL: "https://mock/upload"}, nil
}

func (m *inMemoryMockService) Complete(_ context.Context, id, _ string) (asset.Asset, error) {
	return asset.Asset{ID: id, Status: "completed"}, nil
}

func (m *inMemoryMockService) Fail(id, _ string) (asset.Asset, error) {
	return asset.Asset{ID: id, Status: "failed"}, nil
}

func (m *inMemoryMockService) Content(_ context.Context, _ string) (string, error) {
	return "https://mock/content", nil
}

func TestHTTPAPI_WithMocks(t *testing.T) {
	mock := &inMemoryMockService{}
	mock.state.ID = "test-mock-project"
	mock.state.Revision = 42

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := httpadapter.New(mock, nil, nil, logger)

	// Test GET /api/project
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var state ports.State
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if state.ID != "test-mock-project" || state.Revision != 42 {
		t.Fatalf("unexpected state: %+v", state)
	}
}
