package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"math"
	"mime"
	"path"
	"strings"
	"time"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
)

// saveAsset clones the metadata map, updates the asset, and persists it atomically.
func (s *Service) saveAsset(a asset.Asset) (asset.Asset, error) {
	next := s.meta
	next.Assets = maps.Clone(s.meta.Assets)
	next.Assets[a.ID] = a
	if err := s.catalog.Save(next); err != nil {
		s.blocked = true
		return asset.Asset{}, fmt.Errorf("%w: save metadata: %v", domErrors.ErrUnavailable, err)
	}
	s.meta = next
	return a, nil
}

// Register validates media parameters and creates a new asset entry with "pending" status.
func (s *Service) Register(src asset.Source) (asset.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return asset.Asset{}, domErrors.ErrUnavailable
	}

	src.Name = path.Base(strings.ReplaceAll(strings.TrimSpace(src.Name), "\\", "/"))
	mtype, _, err := mime.ParseMediaType(src.MIME)
	if err != nil || !strings.HasPrefix(mtype, src.Kind+"/") ||
		(src.Kind != "video" && src.Kind != "audio" && src.Kind != "image") ||
		src.Bytes <= 0 || src.Bytes > s.maxUpload || len(src.Name) > 255 ||
		src.Name == "." || src.Name == ".." || src.Name == "/" ||
		strings.ContainsAny(src.Name, "\r\n\x00") || src.Width < 0 || src.Height < 0 ||
		src.Width > 32768 || src.Height > 32768 {
		return asset.Asset{}, fmt.Errorf("%w: invalid media metadata or file size", domErrors.ErrInvalid)
	}
	src.MIME = mtype

	if src.Kind == "image" {
		switch mtype {
		case "image/jpeg", "image/png", "image/webp", "image/gif", "image/avif":
		default:
			return asset.Asset{}, fmt.Errorf("%w: use JPEG, PNG, WebP, GIF, or AVIF images", domErrors.ErrInvalid)
		}
		src.Duration, src.Frames = 5, 150
	} else if src.Duration <= 0 || src.Duration > 86400 || math.IsNaN(src.Duration) ||
		src.Frames != max(1, int(math.Floor(src.Duration*30))) {
		return asset.Asset{}, fmt.Errorf("%w: invalid media duration", domErrors.ErrInvalid)
	}

	id := rand.Text()
	return s.saveAsset(asset.Asset{
		Source:    src,
		ID:        id,
		Bucket:    s.objects.Bucket(),
		ObjectKey: "projects/" + s.meta.ID + "/assets/" + id,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	})
}

// Begin initiates an upload attempt and generates signed POST policy credentials.
func (s *Service) Begin(ctx context.Context, id string) (asset.Asset, asset.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return asset.Asset{}, asset.Upload{}, domErrors.ErrUnavailable
	}

	a, ok := s.meta.Assets[id]
	if !ok {
		return asset.Asset{}, asset.Upload{}, domErrors.ErrNotFound
	}
	if a.Status == "completed" {
		return a, asset.Upload{}, nil
	}

	if a.Attempt != "" {
		if recovered, err := s.recoverFinal(ctx, a); err != nil || recovered.Status == "completed" {
			return recovered, asset.Upload{}, err
		}
	}

	a.Attempt = rand.Text()
	policy, err := s.objects.UploadPolicy(ctx, a.StageKey(), a.Source)
	if err != nil {
		return asset.Asset{}, asset.Upload{}, err
	}

	a.Status = "uploading"
	a, err = s.saveAsset(a)
	return a, policy, err
}

// Complete validates that the staged object matches registered properties and promotes it.
func (s *Service) Complete(ctx context.Context, id, attempt string) (asset.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return asset.Asset{}, domErrors.ErrUnavailable
	}

	a, ok := s.meta.Assets[id]
	if !ok {
		return asset.Asset{}, domErrors.ErrNotFound
	}
	if attempt == "" || attempt != a.Attempt {
		return asset.Asset{}, fmt.Errorf("%w: upload attempt changed", domErrors.ErrConflict)
	}
	if a.Status == "completed" {
		return a, nil
	}

	a, err := s.recoverFinal(ctx, a)
	if err != nil || a.Status == "completed" {
		return a, err
	}

	info, err := s.objects.Stat(ctx, a.StageKey())
	if errors.Is(err, domErrors.ErrNotFound) {
		return asset.Asset{}, fmt.Errorf("%w: upload has not reached MinIO", domErrors.ErrConflict)
	}
	if err != nil {
		return asset.Asset{}, err
	}
	if info.Size != a.Bytes || info.MIME != a.MIME || info.ETag == "" {
		return asset.Asset{}, fmt.Errorf("%w: stored upload does not match registered size and type", domErrors.ErrInvalid)
	}

	if err = s.objects.Promote(ctx, a.StageKey(), a.ObjectKey, info.ETag); err != nil {
		return asset.Asset{}, err
	}
	a.Status = "completed"
	return s.saveAsset(a)
}

// Fail marks an active upload attempt as failed.
func (s *Service) Fail(id, attempt string) (asset.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return asset.Asset{}, domErrors.ErrUnavailable
	}

	a, ok := s.meta.Assets[id]
	if !ok {
		return asset.Asset{}, domErrors.ErrNotFound
	}
	if attempt == "" || attempt != a.Attempt {
		return asset.Asset{}, domErrors.ErrConflict
	}
	if a.Status == "completed" || a.Status == "failed" {
		return a, nil
	}
	a.Status = "failed"
	return s.saveAsset(a)
}

// Content returns a presigned playback/download URL for a completed asset.
func (s *Service) Content(ctx context.Context, id string) (string, error) {
	s.mu.Lock()
	a, ok := s.meta.Assets[id]
	s.mu.Unlock()
	if !ok || a.Status != "completed" {
		return "", domErrors.ErrNotFound
	}
	return s.objects.ReadURL(ctx, a.ObjectKey)
}

// recoverFinal verifies if an earlier attempt promoted the object before metadata was persisted.
func (s *Service) recoverFinal(ctx context.Context, a asset.Asset) (asset.Asset, error) {
	info, err := s.objects.Stat(ctx, a.ObjectKey)
	if errors.Is(err, domErrors.ErrNotFound) {
		return a, nil
	}
	if err != nil {
		return asset.Asset{}, err
	}
	if info.Size != a.Bytes || info.MIME != a.MIME {
		return asset.Asset{}, fmt.Errorf("%w: permanent object metadata does not match", domErrors.ErrConflict)
	}
	a.Status = "completed"
	return s.saveAsset(a)
}
