package ws

import (
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
}

// OutMessage is the envelope for outgoing messages sent to WebSocket clients.
type OutMessage struct {
	Type        string          `json:"type"`
	ReqID       string          `json:"req_id,omitempty"`
	Seq         uint64          `json:"seq,omitempty"`
	Status      string          `json:"status,omitempty"`
	Code        string          `json:"code,omitempty"`
	Message     string          `json:"message,omitempty"`
	CurrentSeq  uint64          `json:"current_seq,omitempty"`
	Op          *timeline.Edit  `json:"op,omitempty"`
	Author      string          `json:"author,omitempty"`
	ClientID    string          `json:"client_id,omitempty"`
	Cursor      *Cursor         `json:"cursor,omitempty"`
	Playhead    *int            `json:"playhead,omitempty"`
	Selected    *string         `json:"selected,omitempty"`
	Asset       *asset.Asset    `json:"asset,omitempty"`
	AssetID     string          `json:"asset_id,omitempty"`
	Ops         []record.Record `json:"ops,omitempty"`
	State       *service.State  `json:"state,omitempty"`
}
