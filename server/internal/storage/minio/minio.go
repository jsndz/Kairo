package minio

import (
	"context"
	"net/url"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

// MinIO implements asset.Objects using the MinIO Go SDK.
type MinIO struct {
	internal, public *miniogo.Client
	bucket           string
}

// New creates one client for internal server calls and one for presigning public URLs.
func New(endpoint, public, access, secret, bucket string, tls bool) (*MinIO, error) {
	options := &miniogo.Options{
		Creds:        credentials.NewStaticV4(access, secret, ""),
		Secure:       tls,
		Region:       "us-east-1",
		BucketLookup: miniogo.BucketLookupPath,
	}
	client, err := miniogo.New(endpoint, options)
	if err != nil {
		return nil, err
	}
	signer, err := miniogo.New(public, options)
	if err != nil {
		return nil, err
	}
	return &MinIO{client, signer, bucket}, nil
}

// Bucket returns the configured media bucket name.
func (m *MinIO) Bucket() string { return m.bucket }

// Ensure verifies or creates the bucket and configures 24-hour staging TTL lifecycle rules.
func (m *MinIO) Ensure(ctx context.Context) error {
	exists, err := m.internal.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err = m.internal.MakeBucket(ctx, m.bucket, miniogo.MakeBucketOptions{}); err != nil {
			return err
		}
	}

	config, err := m.internal.GetBucketLifecycle(ctx, m.bucket)
	if err != nil && miniogo.ToErrorResponse(err).Code != "NoSuchLifecycleConfiguration" {
		return err
	}
	if config == nil {
		config = lifecycle.NewConfiguration()
	}
	for _, r := range config.Rules {
		if r.ID == "kairo-staging-expiry" {
			return nil
		}
	}
	config.Rules = append(config.Rules, lifecycle.Rule{
		ID:         "kairo-staging-expiry",
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: "staging/"},
		Expiration: lifecycle.Expiration{Days: 1},
	})
	return m.internal.SetBucketLifecycle(ctx, m.bucket, config)
}

// UploadPolicy generates presigned POST credentials restricted to exact size and MIME type.
func (m *MinIO) UploadPolicy(ctx context.Context, key string, src asset.Source) (asset.Upload, error) {
	p := miniogo.NewPostPolicy()
	for _, err := range []error{
		p.SetBucket(m.bucket),
		p.SetKey(key),
		p.SetExpires(time.Now().UTC().Add(30 * time.Minute)),
		p.SetContentLengthRange(src.Bytes, src.Bytes),
		p.SetContentType(src.MIME),
	} {
		if err != nil {
			return asset.Upload{}, err
		}
	}
	u, fields, err := m.public.PresignedPostPolicy(ctx, p)
	if err != nil {
		return asset.Upload{}, err
	}
	return asset.Upload{URL: u.String(), Fields: fields}, nil
}

// Stat retrieves object metadata (size, MIME, ETag) without reading object data.
func (m *MinIO) Stat(ctx context.Context, key string) (asset.ObjectInfo, error) {
	i, err := m.internal.StatObject(ctx, m.bucket, key, miniogo.StatObjectOptions{})
	switch miniogo.ToErrorResponse(err).Code {
	case "NoSuchKey", "NotFound":
		return asset.ObjectInfo{}, domErrors.ErrNotFound
	}
	return asset.ObjectInfo{Size: i.Size, MIME: i.ContentType, ETag: i.ETag}, err
}

// Promote copies an object from staging to its permanent immutable key.
func (m *MinIO) Promote(ctx context.Context, stage, key, etag string) error {
	_, err := m.internal.CopyObject(ctx, miniogo.CopyDestOptions{Bucket: m.bucket, Object: key},
		miniogo.CopySrcOptions{Bucket: m.bucket, Object: stage, MatchETag: etag})
	return err
}

// ReadURL checks object existence and returns a 1-hour presigned GET URL for streaming.
func (m *MinIO) ReadURL(ctx context.Context, key string) (string, error) {
	if _, err := m.Stat(ctx, key); err != nil {
		return "", err
	}
	u, err := m.public.PresignedGetObject(ctx, m.bucket, key, time.Hour, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
