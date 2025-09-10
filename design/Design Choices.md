---
# Kairo System Design Notes

This document consolidates the system design choices for Kairo.
---

## 1. Document Persistence

### Tables

```go
type Document struct {
    ID            uint32   `db:"id"`
    Title         string   `db:"title"`
    UserID        uint32   `db:"user_id"`
    CurrentState  []byte   `db:"current_state"` // Yjs snapshot
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}

// Append-only updates / deltas
type DocumentUpdate struct {
    ID          uint32    `db:"id"`
    DocID       uint32    `db:"doc_id"`
    UpdateState []byte    `db:"update_state"` // Yjs delta
    CreatedAt   time.Time `db:"created_at"`
}
```

### Persistence Strategies

- Insert each edit into `document_update`
- Periodically (every few seconds/idle) merge into `current_state`
- Pros: fast writes, full history, reasonably up-to-date reads

### Autosave & Latency

- Writes to `document_update` are cheap (1–5 KB typical delta).
- Append-only writes are fast, <1ms per insert.
- Autosave/compaction every few seconds prevents lag.

---

## 2. WebSocket vs REST for Loading and Saving

### Loading Document Data

- **Load via Gateway (REST)** is preferred:
  - `GET /documents/:id` → gRPC → `doc-service` → DB
  - Clean separation: REST for initial snapshot, WS for real-time updates

### Saving / Updating Metadata

- **Content updates**: sent over WebSocket → persisted in `document_update`
- **Metadata (e.g., rename)**: use separate REST/gRPC endpoint
  - Example: `PATCH /documents/:id/name` or gRPC `RenameDocument`
  - WebSocket not needed for metadata
- **Response design**:
  - Do **not** return full `current_state` for metadata updates (payload can be large)
  - Return only changed fields and optionally version info

---

## 3. CRDT Deltas & Applying Updates

### What are deltas?

- Small, commutative operations describing a document change
- Examples: insert text, delete text, formatting changes
- Stored as byte array in `document_update.update_state`
- In Yjs: use `encodeStateAsUpdate()`

### Applying deltas

```go
doc := NewYDoc()
doc.ApplyUpdate(current_state_from_db)  // snapshot
for _, delta := range get_updates_since_snapshot(doc_id) {
    doc.ApplyUpdate(delta.UpdateState)
}
final_state := doc.EncodeStateAsUpdate()
```

- New clients fetch `current_state` + deltas → full up-to-date state
- Autosave merges deltas into `current_state` periodically

---

## 4. Updating `document_update`

**Update via WebSocket service (default)**

- Client → WS → ws-relay-service → gRPC → doc-service → DB
- Simple, low-latency, good for small-medium apps

---

## 5. General Best Practices

Not implemented Yet

- Keep **metadata separate from content**; only sync content over WebSocket
- **Do not return full **`` for simple metadata updates
- Use **versioning** in responses to prevent conflicts
- Batch or throttle high-frequency edits if DB load grows
- Index `doc_id` in `document_update` for fast reads
- Archive or compact old deltas after snapshot to control table size

---

This document captures the discussion of:

- Persistence strategies
- Metadata vs content handling
- CRDT deltas and reconstruction
- Update flows and architecture choices
- Real-world scaling and latency considerations

---
