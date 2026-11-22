package ws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"server/internal/domain/asset"
	"server/internal/domain/timeline"
	"server/internal/service"
	"server/internal/storage/catalog"
	"server/internal/storage/jsonl"
	"server/internal/ws"
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

func setupTestHub(t *testing.T) (*ws.Hub, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	log := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}

	svc, err := service.Open(cat, log, mockObjects{}, 10<<20)
	if err != nil {
		t.Fatalf("Open service: %v", err)
	}

	hub := ws.NewHub(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/ws", hub.ServeWs)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return hub, server
}

func TestRoomCapacity(t *testing.T) {
	_, server := setupTestHub(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/project/ws"

	// Client 1 connects
	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 1 connect: %v", err)
	}
	defer c1.Close()

	// Client 2 connects
	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 2 connect: %v", err)
	}
	defer c2.Close()

	// Client 3 should be rejected (max 2 users)
	c3, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		c3.Close()
		t.Fatal("Client 3 should have been rejected from room, but connected successfully")
	}
	if resp != nil && resp.StatusCode == http.StatusOK {
		// Gorilla may upgrade then immediately close with ClosePolicyViolation
	}
}

func TestPresenceAndOperations(t *testing.T) {
	_, server := setupTestHub(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/project/ws"

	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 1: %v", err)
	}
	defer c1.Close()

	var init1 ws.OutMessage
	if err := c1.ReadJSON(&init1); err != nil {
		t.Fatalf("Read init1: %v", err)
	}
	if init1.Type != "init" {
		t.Fatalf("Expected init message, got %s", init1.Type)
	}

	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 2: %v", err)
	}
	defer c2.Close()

	var init2 ws.OutMessage
	if err := c2.ReadJSON(&init2); err != nil {
		t.Fatalf("Read init2: %v", err)
	}
	if init2.Type != "init" {
		t.Fatalf("Expected init2, got %s", init2.Type)
	}

	// c1 receives peer_joined notification
	var peerJoin ws.OutMessage
	if err := c1.ReadJSON(&peerJoin); err != nil {
		t.Fatalf("Read peer_joined: %v", err)
	}
	if peerJoin.Type != "peer_joined" {
		t.Fatalf("Expected peer_joined, got %s", peerJoin.Type)
	}

	// c2 receives peer_joined notification about existing c1
	var peerJoin2 ws.OutMessage
	if err := c2.ReadJSON(&peerJoin2); err != nil {
		t.Fatalf("Read peerJoin2: %v", err)
	}
	if peerJoin2.Type != "peer_joined" || peerJoin2.ClientID != init1.ClientID {
		t.Fatalf("Expected peer_joined for c1, got %+v", peerJoin2)
	}

	// c1 sends presence
	playhead := 42
	err = c1.WriteJSON(ws.InMessage{
		Type:     "presence",
		Playhead: &playhead,
		Cursor:   &ws.Cursor{X: 100, Y: 200},
	})
	if err != nil {
		t.Fatalf("Send presence: %v", err)
	}

	// c2 should receive presence_broadcast
	var presence ws.OutMessage
	if err := c2.ReadJSON(&presence); err != nil {
		t.Fatalf("Read presence: %v", err)
	}
	if presence.Type != "presence_broadcast" || *presence.Playhead != 42 {
		t.Fatalf("Unexpected presence message: %+v", presence)
	}

	// c1 submits an operation (rename project)
	expectedRev := uint64(0)
	err = c1.WriteJSON(ws.InMessage{
		Type:  "op_submit",
		ReqID: "req-1",
		Op: &timeline.Edit{
			ID:               "op-1",
			Type:             "rename",
			Payload:          []byte(`{"name":"New Title"}`),
			ExpectedRevision: &expectedRev,
		},
		ExpectedRevision: &expectedRev,
	})
	if err != nil {
		t.Fatalf("Submit op: %v", err)
	}

	// c1 receives op_ack
	var ack ws.OutMessage
	if err := c1.ReadJSON(&ack); err != nil {
		t.Fatalf("Read op_ack: %v", err)
	}
	if ack.Type != "op_ack" || ack.Status != "ok" || ack.Seq != 1 {
		t.Fatalf("Unexpected ack: %+v", ack)
	}

	// c2 receives op_broadcast
	var bcast ws.OutMessage
	if err := c2.ReadJSON(&bcast); err != nil {
		t.Fatalf("Read op_broadcast: %v", err)
	}
	if bcast.Type != "op_broadcast" || bcast.Seq != 1 || bcast.State == nil || bcast.State.Project.Name != "New Title" {
		t.Fatalf("Unexpected op_broadcast: %+v", bcast)
	}
}
