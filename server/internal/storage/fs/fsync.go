package fs

import "os"

// SyncDir performs an fsync on a directory handle, ensuring that directory metadata
// (such as new file entries and atomic renames) is flushed durably to disk.
func SyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
