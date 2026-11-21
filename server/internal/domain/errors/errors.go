package errors

import "errors"

// Sentinel errors representing domain-level error conditions.
var (
	// ErrInvalid indicates malformed requests, invalid payloads, or timeline rule violations.
	ErrInvalid = errors.New("invalid request")

	// ErrConflict indicates a stale revision, duplicate operation ID with differing payload, or upload conflict.
	ErrConflict = errors.New("conflict")

	// ErrNotFound indicates that a requested asset, clip, or object was not found.
	ErrNotFound = errors.New("not found")

	// ErrUnavailable indicates that underlying storage (disk, MinIO) is unreadable or unwritable.
	ErrUnavailable = errors.New("storage unavailable")
)
