package service

// import (
// 	"testing"

// 	"github.com/jsndz/kairo/pkg/db"
// )

// func TestAutoSave(t *testing.T) {
// 	dsn := "postgresql://postgres:kairo@localhost:5434/kairo_doc"
// 	if dsn == "" {
// 		t.Skip("DOC_DB_URL not set, skipping integration test")
// 	}

// 	database, err := db.InitDB(dsn)
// 	if err != nil {
// 		t.Fatalf("failed to connect db: %v", err)
// 	}

// 	s := NewDocService(database)
// 	doc, err := s.Save(1)
// 	if err != nil {
// 		t.Fatalf("AutoSave returned error: %v", err)
// 	}

// 	if doc == nil {
// 		t.Fatalf("expected doc, got nil")
// 	}
// 	if doc.ID != 1 {
// 		t.Errorf("expected doc ID 1, got %d", doc.ID)
// 	}
// }
