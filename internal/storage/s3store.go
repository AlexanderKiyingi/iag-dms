package storage

import (
	"github.com/alvor-technologies/iag-platform-go/objectstore"
)

// NewS3Store returns a Store backed by an S3-compatible bucket, or nil if the
// configuration is incomplete. A nil return is the signal to fall back to disk.
//
// The implementation lives in platform-go/objectstore. It was briefly duplicated
// here - this service got the first copy of contract-management's presigner, and
// procurement and finance were about to make a third and fourth - so it was
// promoted to the shared module instead. objectstore.S3Store satisfies this
// package's Store interface directly: same three methods, same signatures.
//
// This wrapper is kept so call sites and main.go continue to speak to
// internal/storage rather than reaching into platform-go, and so the disk
// fallback stays selected here.
func NewS3Store(endpoint, region, bucket, accessKey, secretKey string, useSSL bool) Store {
	if s := objectstore.NewS3Store(endpoint, region, bucket, accessKey, secretKey, useSSL); s != nil {
		return s
	}
	// A typed nil would make the Store interface non-nil and defeat the caller's
	// nil check, so return an untyped nil.
	return nil
}
