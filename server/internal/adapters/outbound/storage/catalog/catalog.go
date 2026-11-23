package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"

	"server/internal/domain/record"
	"server/internal/adapters/outbound/storage/fs"
)

// JSONCatalog implements record.Catalog using an atomic file-replacement pattern.
type JSONCatalog struct {
	Path string
}

// Read parses the project metadata and asset catalog from disk.
func (c JSONCatalog) Read() (record.Metadata, error) {
	var result record.Metadata
	b, err := os.ReadFile(c.Path)
	if err == nil {
		err = json.Unmarshal(b, &result)
	}
	return result, err
}

// Save writes metadata atomically by creating a temporary file in the same directory,
// syncing it to disk, renaming it over the destination file, and syncing the directory.
func (c JSONCatalog) Save(data record.Metadata) error {
	dir := filepath.Dir(c.Path)
	f, err := os.CreateTemp(dir, ".catalog-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), c.Path); err != nil {
		return err
	}
	return fs.SyncDir(dir)
}
