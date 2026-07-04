// Package storage provides binary object storage for DMS attachments
// (retail-execution photos, document uploads). The default implementation
// writes to the local filesystem; the Store interface is deliberately small so
// an S3/GCS backend can be dropped in later without touching call sites.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Store persists and retrieves opaque objects keyed by a caller-supplied key.
type Store interface {
	// Put streams r to the object identified by key and returns the total bytes
	// written. contentType is advisory (persisted by richer backends).
	Put(key, contentType string, r io.Reader) (int64, error)
	// Open returns a reader for the object; caller must Close it.
	Open(key string) (io.ReadCloser, error)
	// Delete removes the object; missing objects are not an error.
	Delete(key string) error
}

// DiskStore keeps objects under a base directory. Keys may contain forward
// slashes to create sub-directories; path traversal is rejected.
type DiskStore struct {
	base string
}

// NewDiskStore roots a store at dir, creating it if needed.
func NewDiskStore(dir string) (*DiskStore, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/attachments"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: mkdir %s: %w", dir, err)
	}
	return &DiskStore{base: dir}, nil
}

func (d *DiskStore) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + key) // force absolute, collapse .. then strip root
	full := filepath.Join(d.base, clean)
	// Ensure the resolved path stays within base.
	rel, err := filepath.Rel(d.base, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return full, nil
}

func (d *DiskStore) Put(key, _ string, r io.Reader) (int64, error) {
	full, err := d.resolve(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return 0, err
	}
	f, err := os.Create(full)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

func (d *DiskStore) Open(key string) (io.ReadCloser, error) {
	full, err := d.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (d *DiskStore) Delete(key string) error {
	full, err := d.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
