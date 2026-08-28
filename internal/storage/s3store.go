package storage

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"

	"github.com/iag/dms/backend/internal/objstore"
)

// presignExpiry bounds how long a signed URL is valid. These URLs are used
// immediately by this process and never handed to a client, so the window is
// short: long enough to survive a slow upload, short enough that a URL captured
// from a log is useless by the time anyone reads it.
const presignExpiry = 15 * time.Minute

// S3Store implements Store against an S3-compatible bucket (AWS S3, Cloudflare
// R2, MinIO).
//
// It signs a URL and then performs the request itself, rather than pulling in an
// AWS SDK: the presigner already implements SigV4 with the standard library, and
// this keeps DMS free of a large dependency for three verbs.
//
// The alternative was contract-management's model, where the browser is handed a
// presigned URL and uploads directly. That is the better shape for large files,
// but it changes the handler contract; DiskStore's call sites stream bytes
// through Store, and this drops in behind them without touching one of them.
type S3Store struct {
	pre    *objstore.Presigner
	client *http.Client
}

// NewS3Store returns a Store backed by the bucket, or nil if the presigner is
// unconfigured. A nil return is the signal to fall back to disk.
func NewS3Store(endpoint, region, bucket, accessKey, secretKey string, useSSL bool) *S3Store {
	pre := objstore.New(endpoint, region, bucket, accessKey, secretKey, useSSL)
	if pre == nil {
		return nil
	}
	return &S3Store{
		pre: pre,
		// A timeout is deliberate: without one a hung bucket blocks the request
		// goroutine indefinitely, which is how a storage outage becomes a
		// service outage.
		client: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (s *S3Store) Put(key, contentType string, r io.Reader) (int64, error) {
	// The object is counted as it streams, so the byte count is what the bucket
	// actually received rather than what the caller claimed.
	counter := &countingReader{r: r}

	req, err := http.NewRequest(http.MethodPut, s.pre.PresignPut(key, presignExpiry), counter)
	if err != nil {
		return 0, fmt.Errorf("s3 put %s: %w", key, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("s3 put %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return 0, fmt.Errorf("s3 put %s: unexpected status %s", key, resp.Status)
	}
	return counter.n, nil
}

func (s *S3Store) Open(key string) (io.ReadCloser, error) {
	resp, err := s.client.Get(s.pre.PresignGet(key, presignExpiry))
	if err != nil {
		return nil, fmt.Errorf("s3 open %s: %w", key, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("s3 open %s: %w", key, fs.ErrNotExist)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		resp.Body.Close()
		return nil, fmt.Errorf("s3 open %s: unexpected status %s", key, resp.Status)
	}
	return resp.Body, nil
}

// Delete removes the object. A missing object is not an error, matching
// DiskStore and the Store contract.
func (s *S3Store) Delete(key string) error {
	req, err := http.NewRequest(http.MethodDelete, s.pre.PresignDelete(key, presignExpiry), nil)
	if err != nil {
		return fmt.Errorf("s3 delete %s: %w", key, err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete %s: %w", key, err)
	}
	defer resp.Body.Close()
	// S3 returns 204 for a successful delete and, for most implementations, also
	// 204 when the key was never there. 404 is tolerated for the ones that do not.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("s3 delete %s: unexpected status %s", key, resp.Status)
	}
	return nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
