package concensus

import (
	"server/internal/consensus"
)

// Coordinator forwards to consensus.Coordinator.
type Coordinator = consensus.Coordinator

// Proposal forwards to consensus.Proposal.
type Proposal = consensus.Proposal

// New creates a new Coordinator.
var New = consensus.New

// NewCoordinator creates a new Coordinator.
var NewCoordinator = consensus.NewCoordinator
