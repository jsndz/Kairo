package utils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
)

func jsonToBytes(jsonStr string) ([]byte, error) {
	var m map[string]int
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return nil, err
	}

	b := make([]byte, len(m))
	for i := 0; i < len(m); i++ {
		b[i] = byte(m[fmt.Sprint(i)])
	}
	return b, nil
}

func TestCombineDeltaState(t *testing.T) {

var deltas []model.DocumentUpdate
	rawUpdates:= []string{
		`{"0":1,"1":1,"2":0,"3":0,"4":0,"5":0,"6":0}`,
		` {"0":1,"1":3,"2":175,"3":157,"4":164,"5":236,"6":5,"7":0,"8":7,"9":1,"10":7,"11":100,"12":101,"13":102,"14":97,"15":117,"16":108,"17":116,"18":3,"19":9,"20":112,"21":97,"22":114,"23":97,"24":103,"25":114,"26":97,"27":112,"28":104,"29":7,"30":0,"31":175,"32":157,"33":164,"34":236,"35":5,"36":0,"37":6,"38":4,"39":0,"40":175,"41":157,"42":164,"43":236,"44":5,"45":1,"46":1,"47":102,"48":0}`,
		` {"0":1,"1":1,"2":175,"3":157,"4":164,"5":236,"6":5,"7":3,"8":132,"9":175,"10":157,"11":164,"12":236,"13":5,"14":2,"15":1,"16":115,"17":0}`,
		` {"0":1,"1":1,"2":175,"3":157,"4":164,"5":236,"6":5,"7":4,"8":132,"9":175,"10":157,"11":164,"12":236,"13":5,"14":3,"15":1,"16":101,"17":0}`,
		` {"0":1,"1":1,"2":175,"3":157,"4":164,"5":236,"6":5,"7":5,"8":132,"9":175,"10":157,"11":164,"12":236,"13":5,"14":4,"15":1,"16":100,"17":0}`,
		` {"0":1,"1":1,"2":175,"3":157,"4":164,"5":236,"6":5,"7":6,"8":132,"9":175,"10":157,"11":164,"12":236,"13":5,"14":5,"15":1,"16":102,"17":0}`,
	}
	current := []byte(``)
	for _, r := range rawUpdates {
		b, err := jsonToBytes(r)
		if err != nil {
			panic(err)
		}
		deltas = append(deltas, model.DocumentUpdate{UpdateState: b})
	}
	newState, err := CombineDeltaState(current, &deltas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(newState) == 0 {
		t.Fatalf("expected merged state, got empty result")
	}

	t.Logf("merged state: %s", string(newState))
}