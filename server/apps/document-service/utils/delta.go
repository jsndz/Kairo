package utils

import (
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	y_crdt "github.com/skyterra/y-crdt"
)


func CombineDeltaState(current_state []byte,deltas *[]model.DocumentUpdate) ( []byte,error )  {
	var updates [][]byte
	for _, d := range *deltas {
		updates = append(updates, d.UpdateState)
	}
	newState := y_crdt.MergeUpdates(
		updates,
		func(b []byte) *y_crdt.UpdateDecoderV1 {
			return y_crdt.NewUpdateDecoderV1(b)
		},
		func() *y_crdt.UpdateEncoderV1 {
			return y_crdt.NewUpdateEncoderV1()
		},
		false,
	)
	return newState,nil
}