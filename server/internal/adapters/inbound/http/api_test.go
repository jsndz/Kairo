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
	"server/internal/app/project"
	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/ports"
)

type mockObjects struct{}

func (mockObjects) Bucket() string { return "test-bucket" }
func (mockObjects) UploadPolicy(context.Context, string, asset.Source) (asset.Upload, error) {
	return asset.Upload{URL: "https://example.com/put"}, nil
}
func (mockObjects) Stat(context.Context, string) (asset.ObjectInfo, error) {
	return asset.ObjectInfo{Size: 100, MIME: "video/mp4"}, nil
}
func (mockObjects) Promote(context.Context, string, string, string) error { return nil }
func (mockObjects) ReadURL(context.Context, string) (string, error) {
	return "https://example.com/get", nil
}
func (mockObjects) Download(context.Context, string, string) error { return nil }
func (mockObjects) Upload(context.Context, string, string, string) error { return nil }

func setupAPI(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	obj := mockObjects{}

	mgr, err := project.NewManager(dir, obj, 10<<20)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	summary, _, err := mgr.Create(context.Background(), "Test Project")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httpadapter.New(mgr, nil, nil, logger), summary.ID
}

func TestHealthCheck(t *testing.T) {
	handler, _ := setupAPI(t)
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

func TestGetProjectByID(t *testing.T) {
	handler, projID := setupAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+projID, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var state ports.State
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("failed to decode project state: %v", err)
	}
	if state.ID != projID {
		t.Fatalf("expected project ID %q, got %q", projID, state.ID)
	}
}

func TestAssetRegistrationAndValidation(t *testing.T) {
	handler, projID := setupAPI(t)

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
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projID+"/assets", bytes.NewReader(bodyBytes))
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
	reqBad := httptest.NewRequest(http.MethodPost, "/api/projects/"+projID+"/assets", bytes.NewReader(badBytes))
	reqBad.Header.Set("Content-Type", "application/json")
	recBad := httptest.NewRecorder()

	handler.ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid mime, got %d", recBad.Code)
	}

	// Content-Type enforcement
	reqBadCT := httptest.NewRequest(http.MethodPost, "/api/projects/"+projID+"/assets", bytes.NewReader(bodyBytes))
	reqBadCT.Header.Set("Content-Type", "text/plain")
	recBadCT := httptest.NewRecorder()
	handler.ServeHTTP(recBadCT, reqBadCT)
	if recBadCT.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for bad Content-Type, got %d", recBadCT.Code)
	}

	// Cross-site fetch protection
	reqCSRF := httptest.NewRequest(http.MethodPost, "/api/projects/"+projID+"/assets", bytes.NewReader(bodyBytes))
	reqCSRF.Header.Set("Content-Type", "application/json")
	reqCSRF.Header.Set("Sec-Fetch-Site", "cross-site")
	recCSRF := httptest.NewRecorder()
	handler.ServeHTTP(recCSRF, reqCSRF)
	if recCSRF.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for cross-site post, got %d", recCSRF.Code)
	}
}

type mockManagerForRoomHTTP struct{}

func (m *mockManagerForRoomHTTP) List(_ context.Context) ([]record.ProjectSummary, error) {
	return nil, nil
}
func (m *mockManagerForRoomHTTP) Create(_ context.Context, name string) (record.ProjectSummary, ports.ProjectInstance, error) {
	return record.ProjectSummary{ID: "room-generated-999", Name: name}, nil, nil
}
func (m *mockManagerForRoomHTTP) Get(_ context.Context, _ string) (ports.ProjectInstance, error) {
	return nil, nil
}
func (m *mockManagerForRoomHTTP) Delete(_ context.Context, _ string) error {
	return nil
}
func (m *mockManagerForRoomHTTP) Touch(_ string) {}

func TestCreateRoom(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mgr := &mockManagerForRoomHTTP{}
	handler := httpadapter.New(mgr, nil, nil, logger)

	req := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"name":"My Edit Room"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var res map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res["roomId"] != "room-generated-999" || res["name"] != "My Edit Room" {
		t.Fatalf("unexpected room created: %+v", res)
	}
}
