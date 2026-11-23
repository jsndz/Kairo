package ws

import (
	"server/internal/app/collaboration"
)

// Cursor represents cursor coordinates on the timeline.
type Cursor = collaboration.Cursor

// InMessage is the envelope for incoming messages from a WebSocket client.
type InMessage = collaboration.InMessage

// OutMessage is the envelope for outgoing messages sent to WebSocket clients.
type OutMessage = collaboration.OutMessage
