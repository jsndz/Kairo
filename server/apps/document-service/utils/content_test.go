package utils

import (
	"testing"

	y "github.com/skyterra/y-crdt"
)




func TestCheckContent(t *testing.T) {
    doc1 := y.NewDoc("default", true, nil, nil, false)
    text := doc1.GetText("content")
    text.Insert(0, "Hello CRDT!", nil)

    update := y.EncodeStateAsUpdate(doc1, nil)

    newState := GetContent(update)

    if newState == "" {
        t.Fatalf("expected valid state, got nil")
    }
    t.Logf("content state: %s", newState)
}