package ports

import (
	"time"

	"server/internal/domain/asset"
	"server/internal/domain/timeline"
)

// Proposal represents an active consensus proposal.
type Proposal struct {
	ID          string          `json:"id"`
	ProposerID  string          `json:"proposerId"`
	Op          timeline.Edit   `json:"op"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExpiresAt   time.Time       `json:"expiresAt"`
	Votes       map[string]bool `json:"votes"` // clientID -> approve
}

// CollaborationSession defines the transport-agnostic room session interface.
type CollaborationSession interface {
	CanJoin(clientID string) bool
	Join(clientID string) error
	Leave(clientID string)
	SetGraceDuration(d time.Duration)
	BroadcastAsset(a asset.Asset)
	BroadcastAssetDeleted(assetID string)
	BroadcastOperation(projectID string, edit timeline.Edit, state State, author string)
}

// ConsensusCoordinator defines the proposal and voting coordination interface.
type ConsensusCoordinator interface {
	Propose(proposalID, proposerID string, op timeline.Edit, description string, timeout time.Duration) (*Proposal, error)
	Vote(proposalID, voterID string, approve bool) (resolved bool, approved bool, op *timeline.Edit, err error)
}
