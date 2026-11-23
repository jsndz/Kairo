package jsonl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"server/internal/domain/record"
)

// MaxRecordBytes sets a hard limit on an individual JSONL operation record (1 MiB).
const MaxRecordBytes = 1 << 20

// JSONL implements record.Log using an append-only JSON Lines file.
type JSONL struct {
	Path string
}

// Read loads operation records sequentially. If the file ends with an incomplete line
// (e.g. from an ungraceful shutdown or crash mid-write), it truncates the partial row safely.
func (l JSONL) Read() ([]record.Record, error) {
	f, err := os.OpenFile(l.Path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var records []record.Record
	var offset int64

	for {
		var row []byte
		for {
			part, readErr := reader.ReadSlice('\n')
			if len(row)+len(part) > MaxRecordBytes {
				return nil, fmt.Errorf("operation record %d exceeds 1 MiB", len(records)+1)
			}
			row = append(row, part...)
			err = readErr
			if !errors.Is(err, bufio.ErrBufferFull) {
				break
			}
		}

		if err == io.EOF {
			// If there was an uncompleted line at EOF, truncate it to repair the log file
			if len(row) > 0 {
				if err = f.Truncate(offset); err != nil {
					return nil, err
				}
				if err = f.Sync(); err != nil {
					return nil, err
				}
			}
			return records, nil
		}
		if err != nil {
			return nil, err
		}

		var rec record.Record
		if err = json.Unmarshal(row, &rec); err != nil {
			return nil, fmt.Errorf("invalid log line %d: %w", len(records)+1, err)
		}
		records = append(records, rec)
		offset += int64(len(row))
	}
}

// Append serializes an operation record to a single JSON line and flushes it to disk.
func (l JSONL) Append(rec record.Record) error {
	row, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(l.Path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(append(row, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
