package ws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"server/internal/domain/asset"
	"server/internal/domain/timeline"
	"server/internal/app/project"
	"server/internal/adapters/outbound/storage/catalog"
	"server/internal/adapters/outbound/storage/jsonl"
	"server/internal/adapters/inbound/ws"
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

func setupTestHub(t *testing.T) (*ws.Hub, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	cat := catalog.JSONCatalog{Path: filepath.Join(dir, "metadata.json")}
	log := jsonl.JSONL{Path: filepath.Join(dir, "operations.jsonl")}

	svc, err := project.Open(cat, log, mockObjects{}, 10<<20)
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

func connectTwoClients(t *testing.T, server *httptest.Server) (*websocket.Conn, *websocket.Conn, ws.OutMessage, ws.OutMessage) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/project/ws"

	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial c1: %v", err)
	}

	var init1 ws.OutMessage
	if err := c1.ReadJSON(&init1); err != nil {
		t.Fatalf("Read init1: %v", err)
	}

	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial c2: %v", err)
	}

	var init2 ws.OutMessage
	if err := c2.ReadJSON(&init2); err != nil {
		t.Fatalf("Read init2: %v", err)
	}

	// c1 reads peer_joined
	var join1 ws.OutMessage
	_ = c1.ReadJSON(&join1)

	// c2 reads peer_joined
	var join2 ws.OutMessage
	_ = c2.ReadJSON(&join2)

	return c1, c2, init1, init2
}

func TestRoom_ClaimAndRelease(t *testing.T) {
	_, server := setupTestHub(t)
	c1, c2, _, _ := connectTwoClients(t, server)
	defer c1.Close()
	defer c2.Close()

	// 1. c1 claims clip-1 and clip-2
	err := c1.WriteJSON(ws.InMessage{
		Type:    "claim_request",
		ReqID:   "claim-req-1",
		ClipIDs: []string{"clip-1", "clip-2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var ack1 ws.OutMessage
	if err := c1.ReadJSON(&ack1); err != nil {
		t.Fatalf("Read claim_ack: %v", err)
	}
	if ack1.Type != "claim_ack" || !reflect.DeepEqual(ack1.Granted, []string{"clip-1", "clip-2"}) {
		t.Fatalf("Unexpected claim_ack: %+v", ack1)
	}

	// c1 and c2 receive claim_broadcast
	var bcast1, bcast2 ws.OutMessage
	_ = c1.ReadJSON(&bcast1)
	_ = c2.ReadJSON(&bcast2)
	if bcast2.Type != "claim_broadcast" || !reflect.DeepEqual(bcast2.ClipIDs, []string{"clip-1", "clip-2"}) {
		t.Fatalf("Unexpected claim_broadcast on c2: %+v", bcast2)
	}

	// 2. c2 attempts to claim clip-1 (already owned) and clip-3 (unassigned)
	err = c2.WriteJSON(ws.InMessage{
		Type:    "claim_request",
		ReqID:   "claim-req-2",
		ClipIDs: []string{"clip-1", "clip-3"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var ack2 ws.OutMessage
	if err := c2.ReadJSON(&ack2); err != nil {
		t.Fatalf("Read claim_ack c2: %v", err)
	}
	if !reflect.DeepEqual(ack2.Granted, []string{"clip-3"}) || !reflect.DeepEqual(ack2.Rejected, []string{"clip-1"}) {
		t.Fatalf("Unexpected claim_ack on c2: %+v", ack2)
	}

	// Consume claim_broadcast for clip-3
	_ = c1.ReadJSON(&bcast1)
	_ = c2.ReadJSON(&bcast2)

	// 3. c1 voluntarily releases clip-1
	err = c1.WriteJSON(ws.InMessage{
		Type:    "release_request",
		ClipIDs: []string{"clip-1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var rel1, rel2 ws.OutMessage
	_ = c1.ReadJSON(&rel1)
	_ = c2.ReadJSON(&rel2)
	if rel2.Type != "release_broadcast" || !reflect.DeepEqual(rel2.ClipIDs, []string{"clip-1"}) {
		t.Fatalf("Unexpected release_broadcast: %+v", rel2)
	}
}

func TestRoom_PermissionHandshake(t *testing.T) {
	_, server := setupTestHub(t)
	c1, c2, init1, init2 := connectTwoClients(t, server)
	defer c1.Close()
	defer c2.Close()

	// 1. c1 claims clip-A
	_ = c1.WriteJSON(ws.InMessage{
		Type:    "claim_request",
		ReqID:   "c-1",
		ClipIDs: []string{"clip-A"},
	})
	var ack ws.OutMessage
	_ = c1.ReadJSON(&ack)
	var b1, b2 ws.OutMessage
	_ = c1.ReadJSON(&b1)
	_ = c2.ReadJSON(&b2)

	// 2. c2 asks permission to edit clip-A
	err := c2.WriteJSON(ws.InMessage{
		Type:   "permission_request",
		ReqID:  "perm-1",
		ClipID: "clip-A",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. c1 receives permission_prompt
	var prompt ws.OutMessage
	if err := c1.ReadJSON(&prompt); err != nil {
		t.Fatalf("Read permission_prompt: %v", err)
	}
	if prompt.Type != "permission_prompt" || prompt.ClipID != "clip-A" || prompt.FromClientID != init2.ClientID {
		t.Fatalf("Unexpected permission_prompt: %+v", prompt)
	}

	// 4. c1 approves the transfer
	approved := true
	err = c1.WriteJSON(ws.InMessage{
		Type:     "permission_response",
		ReqID:    "perm-1",
		Approved: &approved,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 5. Both receive permission_resolved with new_owner = c2
	var res1, res2 ws.OutMessage
	_ = c1.ReadJSON(&res1)
	_ = c2.ReadJSON(&res2)
	if res2.Type != "permission_resolved" || res2.Approved == nil || !*res2.Approved || res2.NewOwner == nil || *res2.NewOwner != init2.ClientID {
		t.Fatalf("Unexpected permission_resolved: %+v", res2)
	}

	// Both receive claim_broadcast for clip-A owned by c2
	_ = c1.ReadJSON(&res1)
	_ = c2.ReadJSON(&res2)
	if res2.Type != "claim_broadcast" || res2.ClientID != init2.ClientID || !reflect.DeepEqual(res2.ClipIDs, []string{"clip-A"}) {
		t.Fatalf("Unexpected claim_broadcast: %+v", res2)
	}

	_ = init1
}

func TestRoom_ConsensusProposalAndVote(t *testing.T) {
	_, server := setupTestHub(t)
	c1, c2, _, _ := connectTwoClients(t, server)
	defer c1.Close()
	defer c2.Close()

	// 1. c1 proposes adding a track
	err := c1.WriteJSON(ws.InMessage{
		Type:        "consensus_propose",
		ProposalID:  "prop-track-1",
		Description: "Add Video Track 2",
		Op: &timeline.Edit{
			ID:      "op-track-1",
			Type:    "add-track",
			Payload: []byte(`{"trackId":"video-track-2"}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. c2 receives consensus_prompt
	var prompt ws.OutMessage
	if err := c2.ReadJSON(&prompt); err != nil {
		t.Fatalf("Read consensus_prompt: %v", err)
	}
	if prompt.Type != "consensus_prompt" || prompt.ProposalID != "prop-track-1" {
		t.Fatalf("Unexpected consensus_prompt: %+v", prompt)
	}

	// 3. c2 votes to approve
	approved := true
	err = c2.WriteJSON(ws.InMessage{
		Type:       "consensus_vote",
		ProposalID: "prop-track-1",
		Approved:   &approved,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. Both receive consensus_resolved with approved=true
	var res1, res2 ws.OutMessage
	_ = c1.ReadJSON(&res1)
	_ = c2.ReadJSON(&res2)
	if res2.Type != "consensus_resolved" || res2.Approved == nil || !*res2.Approved {
		t.Fatalf("Unexpected consensus_resolved: %+v", res2)
	}

	// 5. Both receive op_broadcast confirming track committed
	var bcast1, bcast2 ws.OutMessage
	_ = c1.ReadJSON(&bcast1)
	_ = c2.ReadJSON(&bcast2)
	if bcast2.Type != "op_broadcast" || bcast2.State == nil {
		t.Fatalf("Unexpected op_broadcast: %+v", bcast2)
	}

	tracks := bcast2.State.Project.Tracks
	foundTrack := false
	for _, tr := range tracks {
		if tr.ID == "video-track-2" {
			foundTrack = true
			break
		}
	}
	if !foundTrack {
		t.Fatalf("Expected video-track-2 in committed project tracks, got: %+v", tracks)
	}

	// 6. Test delete-track via consensus
	err = c2.WriteJSON(ws.InMessage{
		Type:        "consensus_propose",
		ProposalID:  "prop-del-track-1",
		Description: "Delete Track video-track-2",
		Op: &timeline.Edit{
			ID:      "op-del-track-1",
			Type:    "delete-track",
			Payload: []byte(`{"trackId":"video-track-2"}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// c1 receives prompt
	var delPrompt ws.OutMessage
	if err := c1.ReadJSON(&delPrompt); err != nil {
		t.Fatalf("Read consensus_prompt for delete-track: %v", err)
	}
	if delPrompt.Type != "consensus_prompt" || delPrompt.ProposalID != "prop-del-track-1" {
		t.Fatalf("Unexpected consensus_prompt for delete-track: %+v", delPrompt)
	}

	// c1 approves
	err = c1.WriteJSON(ws.InMessage{
		Type:       "consensus_vote",
		ProposalID: "prop-del-track-1",
		Approved:   &approved,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Both receive consensus_resolved
	_ = c1.ReadJSON(&res1)
	_ = c2.ReadJSON(&res2)
	if res2.Type != "consensus_resolved" || res2.Approved == nil || !*res2.Approved {
		t.Fatalf("Unexpected delete-track consensus_resolved: %+v", res2)
	}

	// Both receive op_broadcast confirming track deleted
	_ = c1.ReadJSON(&bcast1)
	_ = c2.ReadJSON(&bcast2)
	if bcast2.Type != "op_broadcast" || bcast2.State == nil {
		t.Fatalf("Unexpected delete-track op_broadcast: %+v", bcast2)
	}
	for _, tr := range bcast2.State.Project.Tracks {
		if tr.ID == "video-track-2" {
			t.Fatalf("Expected video-track-2 to be deleted, but still found in: %+v", bcast2.State.Project.Tracks)
		}
	}
}

func TestRoom_ConsensusBrowserRawJSON(t *testing.T) {
	_, server := setupTestHub(t)
	c1, c2, _, _ := connectTwoClients(t, server)
	defer c1.Close()
	defer c2.Close()

	// 1. Propose add audio track
	msg := `{"type":"consensus_propose","proposal_id":"prop-audio-1","op":{"id":"prop-audio-1","type":"add-track","payload":{"trackId":"audio-2","kind":"audio"}},"description":"Add Audio Track"}`
	if err := c1.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatal(err)
	}

	var prompt ws.OutMessage
	if err := c2.ReadJSON(&prompt); err != nil {
		t.Fatalf("Read prompt: %v", err)
	}
	t.Logf("Prompt received: %+v", prompt)
	if prompt.Type != "consensus_prompt" {
		t.Fatalf("Expected consensus_prompt, got %s", prompt.Type)
	}

	// 2. Vote approve
	voteMsg := `{"type":"consensus_vote","proposal_id":"prop-audio-1","approved":true}`
	if err := c2.WriteMessage(websocket.TextMessage, []byte(voteMsg)); err != nil {
		t.Fatal(err)
	}

	var res1, res2 ws.OutMessage
	_ = c1.ReadJSON(&res1)
	_ = c2.ReadJSON(&res2)
	t.Logf("Resolved res1: %+v, res2: %+v", res1, res2)
	if res2.Type != "consensus_resolved" || res2.Approved == nil || !*res2.Approved {
		t.Fatalf("Expected consensus_resolved approved=true, got %+v", res2)
	}

	var bcast1, bcast2 ws.OutMessage
	_ = c1.ReadJSON(&bcast1)
	_ = c2.ReadJSON(&bcast2)
	t.Logf("Broadcast bcast1: %+v, bcast2: %+v", bcast1, bcast2)
	if bcast2.Type != "op_broadcast" || bcast2.State == nil {
		t.Fatalf("Expected op_broadcast, got %+v", bcast2)
	}
}

func TestRoom_OwnershipReclamationOnDisconnect(t *testing.T) {
	hub, server := setupTestHub(t)
	c1, c2, _, _ := connectTwoClients(t, server)
	defer c2.Close()

	room := hub.GetOrCreateRoom(hub.Service().ProjectID())
	room.SetGraceDuration(50 * time.Millisecond)

	// c1 claims clip-X
	_ = c1.WriteJSON(ws.InMessage{
		Type:    "claim_request",
		ReqID:   "req-x",
		ClipIDs: []string{"clip-X"},
	})
	var ack, b1, b2 ws.OutMessage
	_ = c1.ReadJSON(&ack)
	_ = c1.ReadJSON(&b1)
	_ = c2.ReadJSON(&b2)

	// c1 disconnects
	_ = c1.Close()

	// c2 receives peer_left
	var left ws.OutMessage
	if err := c2.ReadJSON(&left); err != nil {
		t.Fatalf("Read peer_left: %v", err)
	}
	if left.Type != "peer_left" {
		t.Fatalf("Expected peer_left, got %s", left.Type)
	}

	// After grace expiration (50ms), c2 receives release_broadcast for clip-X
	var rel ws.OutMessage
	if err := c2.ReadJSON(&rel); err != nil {
		t.Fatalf("Read release_broadcast: %v", err)
	}
	if rel.Type != "release_broadcast" || !reflect.DeepEqual(rel.ClipIDs, []string{"clip-X"}) {
		t.Fatalf("Unexpected release_broadcast on grace expiration: %+v", rel)
	}
}

func TestRoom_OpSubmitAuthorization(t *testing.T) {
	_, server := setupTestHub(t)
	c1, c2, _, _ := connectTwoClients(t, server)
	defer c1.Close()
	defer c2.Close()

	// c1 claims clip-locked
	_ = c1.WriteJSON(ws.InMessage{
		Type:    "claim_request",
		ReqID:   "req-lock",
		ClipIDs: []string{"clip-locked"},
	})
	var ack, b1, b2 ws.OutMessage
	_ = c1.ReadJSON(&ack)
	_ = c1.ReadJSON(&b1)
	_ = c2.ReadJSON(&b2)

	// c2 tries to delete clip-locked -> rejected with conflict
	expRev := uint64(0)
	err := c2.WriteJSON(ws.InMessage{
		Type:  "op_submit",
		ReqID: "submit-1",
		Op: &timeline.Edit{
			ID:               "op-del",
			Type:             "delete",
			Payload:          []byte(`{"clipId":"clip-locked"}`),
			ExpectedRevision: &expRev,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var opErr ws.OutMessage
	if err := c2.ReadJSON(&opErr); err != nil {
		t.Fatalf("Read op_error: %v", err)
	}
	if opErr.Type != "op_error" || opErr.Code != "conflict" {
		t.Fatalf("Expected conflict op_error, got: %+v", opErr)
	}
}

