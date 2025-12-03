package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
)

// func CombineDeltaState(currentState []byte, deltas *[]model.DocumentUpdate) ([]byte, error) {
// 	doc := y.NewDoc("default", true, nil, nil, false)

// 	if len(currentState) > 2 {
// 		log.Printf("[CombineDeltaState] applying current_state len=%d", len(currentState))
// 		y.ApplyUpdate(doc, currentState, nil)
// 	} else {
// 		log.Printf("[CombineDeltaState] initializing new Y.Doc with text type 'content'")
// 		doc.GetText("content")
// 	}

// 	for i, delta := range *deltas {
// 		if len(delta.UpdateState) == 0 {
// 			log.Printf("Skipping empty update #%d", i)
// 			continue
// 		}

// 		log.Printf("[CombineDeltaState] applying delta #%d len=%d", i, len(delta.UpdateState))
// 		y.ApplyUpdate(doc, delta.UpdateState, nil)
// 	}

// 	finalState := y.EncodeStateAsUpdate(doc, nil)
// 	log.Printf("[CombineDeltaState] merged  deltas → final len=%d", len(finalState))
// 	return finalState, nil
// }

func CombineDeltaState(currentState []byte, deltas *[]model.DocumentUpdate) ([]byte, error) {
	client := &http.Client{}

	body := map[string]any{
		"current_state": base64.StdEncoding.EncodeToString(currentState),
	}

	updates := make([]string, 0, len(*deltas))
	for _, delta := range *deltas {
		if len(delta.UpdateState) > 0 {
			updates = append(updates, base64.StdEncoding.EncodeToString(delta.UpdateState))
		}
	}
	body["updates"] = updates

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merge request: %w", err)
	}

	resp, err := client.Post("http://localhost:3005/merge", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to call merge service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("merge service returned %s: %s", resp.Status, string(body))
	}

	var res struct {
		Merged string `json:"merged"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode merge response: %w", err)
	}

	mergedBytes, err := base64.StdEncoding.DecodeString(res.Merged)
	if err != nil {
		return nil, fmt.Errorf("failed to decode merged state: %w", err)
	}

	return mergedBytes, nil
}
