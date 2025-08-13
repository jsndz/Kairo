package model

import "time"

type Document struct {
    ID            uint32   `db:"id"`
    Title         string    `db:"title"`
    UserID        uint32   `db:"user_id"`
    CurrentState  []byte    `db:"current_state"` 
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}

type DocumentUpdate struct {
    ID          uint32    `db:"id"`
    DocID       uint32    `db:"doc_id"`
    UpdateState []byte    `db:"update_state"` 
    CreatedAt   time.Time `db:"created_at"`
}

