package ws

import (
	"encoding/json"
	"server/internal/domain/asset"
	"server/internal/domain/record"
	"server/internal/domain/timeline"
	"server/internal/service"
)

// Cursor represents cursor coordinates on the timeline.
type Cursor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// InMessage is the envelope for incoming messages from a WebSocket client.
type InMessage struct {
	Type             string         `json:"type"`
	ReqID            string         `json:"req_id,omitempty"`
	Op               *timeline.Edit `json:"op,omitempty"`
	ExpectedRevision *uint64        `json:"expected_revision,omitempty"`
	Cursor           *Cursor        `json:"cursor,omitempty"`
	Playhead         *int           `json:"playhead,omitempty"`
	Selected         *string        `json:"selected,omitempty"`
	LastSeenSeq      *uint64        `json:"last_seen_seq,omitempty"`

	// Stage 3 fields
	ClipIDs     []string `json:"clip_ids,omitempty"`
	ClipID      string   `json:"clip_id,omitempty"`
	Approved    *bool    `json:"approved,omitempty"`
	ProposalID  string   `json:"proposal_id,omitempty"`
	Description string   `json:"description,omitempty"`
}

// UnmarshalJSON handles both nested and flat op formats.
func (m *InMessage) UnmarshalJSON(data []byte) error {
	type InMessageShadow InMessage
	var shadow InMessageShadow
	if err := json.Unmarshal(data, &shadow); err != nil {
		return err
	}
	*m = InMessage(shadow)

	var aux struct {
		RawOp json.RawMessage `json:"op,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err == nil && aux.RawOp != nil {
		if m.Op == nil {
			var edit timeline.Edit
			if err := json.Unmarshal(aux.RawOp, &edit); err == nil {
				m.Op = &edit
			}
		}
		if m.Op != nil && (len(m.Op.Payload) == 0 || string(m.Op.Payload) == "null") {
			var flat map[string]json.RawMessage
			if err := json.Unmarshal(aux.RawOp, &flat); err == nil {
				delete(flat, "id")
				delete(flat, "type")
				delete(flat, "expectedRevision")
				if len(flat) > 0 {
					m.Op.Payload, _ = json.Marshal(flat)
				} else {
					m.Op.Payload = json.RawMessage("{}")
				}
			}
		}
	}
	return nil
}

// OutMessage is the envelope for outgoing messages sent to WebSocket clients.
type OutMessage struct {
	Type         string            `json:"type"`
	ReqID        string            `json:"req_id,omitempty"`
	Seq          uint64            `json:"seq,omitempty"`
	Status       string            `json:"status,omitempty"`
	Code         string            `json:"code,omitempty"`
	Message      string            `json:"message,omitempty"`
	CurrentSeq   uint64            `json:"current_seq,omitempty"`
	Op           *timeline.Edit    `json:"op,omitempty"`
	Author       string            `json:"author,omitempty"`
	ClientID     string            `json:"client_id,omitempty"`
	Cursor       *Cursor           `json:"cursor,omitempty"`
	Playhead     *int              `json:"playhead,omitempty"`
	Selected     *string           `json:"selected,omitempty"`
	Asset        *asset.Asset      `json:"asset,omitempty"`
	AssetID      string            `json:"asset_id,omitempty"`
	Ops          []record.Record   `json:"ops,omitempty"`
	State        *service.State    `json:"state,omitempty"`

	// Stage 3 fields
	Granted      []string          `json:"granted,omitempty"`
	Rejected     []string          `json:"rejected,omitempty"`
	ClipIDs      []string          `json:"clip_ids,omitempty"`
	ClipID       string            `json:"clip_id,omitempty"`
	FromClientID string            `json:"from_client_id,omitempty"`
	TimeoutMS    int               `json:"timeout_ms,omitempty"`
	Approved     *bool             `json:"approved,omitempty"`
	NewOwner     *string           `json:"new_owner,omitempty"`
	ProposalID   string            `json:"proposal_id,omitempty"`
	ProposerID   string            `json:"proposer_id,omitempty"`
	Description  string            `json:"description,omitempty"`
	Owners       map[string]string `json:"owners,omitempty"`
}
