// Package obsstore wraps the minimum Huawei Cloud OBS operations the run
// artifact pipeline needs: streaming upload, streaming read, delete and a
// short-lived presigned GET URL. It is intentionally deployment-agnostic: the
// control plane supplies credentials through the environment only.
package obsstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

// Config is the OBS binding loaded from the process environment. Every
// credential stays in memory; it is never persisted or logged.
type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	// PublicBase optionally overrides the host used in presigned download
	// URLs (for example the bucket's public domain).
	PublicBase string
}

// Enabled reports whether the required OBS fields are present.
func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Endpoint) != "" &&
		strings.TrimSpace(c.Bucket) != "" &&
		strings.TrimSpace(c.AccessKey) != "" &&
		strings.TrimSpace(c.SecretKey) != ""
}

// LoadConfig reads the OBS binding from AGENTFLOW_OBS_* variables.
func LoadConfig() Config {
	return Config{
		Endpoint:   strings.TrimSpace(os.Getenv("AGENTFLOW_OBS_ENDPOINT")),
		Bucket:     strings.TrimSpace(os.Getenv("AGENTFLOW_OBS_BUCKET")),
		AccessKey:  strings.TrimSpace(os.Getenv("AGENTFLOW_OBS_ACCESS_KEY")),
		SecretKey:  strings.TrimSpace(os.Getenv("AGENTFLOW_OBS_SECRET_KEY")),
		PublicBase: strings.TrimSpace(os.Getenv("AGENTFLOW_OBS_PUBLIC_BASE")),
	}
}

// BlobStore is the object-storage seam. A nil store means OBS is not
// configured and callers keep the legacy inline behavior.
type BlobStore interface {
	Enabled() bool
	Put(ctx context.Context, key string, src io.Reader, size int64) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// New builds a BlobStore. A disabled config returns (nil, nil) so callers can
// treat "not configured" as a normal state rather than an error.
func New(cfg Config) (BlobStore, error) {
	if !cfg.Enabled() {
		return nil, nil
	}
	endpoint := normalizeEndpoint(cfg.Endpoint)
	client, err := obs.New(cfg.AccessKey, cfg.SecretKey, endpoint)
	if err != nil {
		return nil, fmt.Errorf("obs client: %w", err)
	}
	return &store{client: client, bucket: cfg.Bucket, publicBase: cfg.PublicBase}, nil
}

func normalizeEndpoint(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	return strings.TrimRight(raw, "/")
}

type store struct {
	client     *obs.ObsClient
	bucket     string
	publicBase string
}

func (s *store) Enabled() bool { return s != nil && s.client != nil }

func (s *store) Put(ctx context.Context, key string, src io.Reader, size int64) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	_, err := s.client.PutObject(&obs.PutObjectInput{
		PutObjectBasicInput: obs.PutObjectBasicInput{
			ObjectOperationInput: obs.ObjectOperationInput{Bucket: s.bucket, Key: key},
			ContentLength:        size,
		},
		Body: src,
	})
	if err != nil {
		return fmt.Errorf("obs put %s: %w", key, err)
	}
	return nil
}

func (s *store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	out, err := s.client.GetObject(&obs.GetObjectInput{
		GetObjectMetadataInput: obs.GetObjectMetadataInput{Bucket: s.bucket, Key: key},
	})
	if err != nil {
		return nil, fmt.Errorf("obs get %s: %w", key, err)
	}
	return out.Body, nil
}

func (s *store) Delete(ctx context.Context, key string) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	if _, err := s.client.DeleteObject(&obs.DeleteObjectInput{Bucket: s.bucket, Key: key}); err != nil {
		return fmt.Errorf("obs delete %s: %w", key, err)
	}
	return nil
}

func (s *store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if !s.Enabled() {
		return "", ErrDisabled
	}
	seconds := int(ttl.Seconds())
	if seconds <= 0 {
		seconds = int((5 * time.Minute).Seconds())
	}
	out, err := s.client.CreateSignedUrl(&obs.CreateSignedUrlInput{
		Method:  obs.HTTP_GET,
		Bucket:  s.bucket,
		Key:     key,
		Expires: seconds,
	})
	if err != nil {
		return "", fmt.Errorf("obs presign %s: %w", key, err)
	}
	return applyPublicBase(out.SignedUrl, s.publicBase), nil
}

// ErrDisabled is returned when an operation reaches a store that was never
// configured. Callers normally guard on Enabled before doing so.
var ErrDisabled = fmt.Errorf("obs store is not configured")

// ArtifactKey is the deterministic object key for one run artifact.
func ArtifactKey(runID, nodeID string, attempt int, rel string) string {
	return path.Join("runs", SafeSegment(runID), SafeSegment(nodeID), fmt.Sprintf("%d", attempt), SafeKey(rel))
}

// SafeSegment keeps a single path segment free of separators and dots.
func SafeSegment(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\x00", "")
	raw = strings.ReplaceAll(raw, "/", "-")
	raw = strings.ReplaceAll(raw, "\\", "-")
	raw = strings.Trim(raw, ".")
	if raw == "" {
		return "item"
	}
	return raw
}

// SafeKey keeps a workspace-relative path as an object key while removing
// traversal and control characters.
func SafeKey(raw string) string {
	parts := strings.Split(strings.ReplaceAll(raw, "\\", "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		part = strings.ReplaceAll(part, "\x00", "")
		out = append(out, part)
	}
	if len(out) == 0 {
		return "item"
	}
	return strings.Join(out, "/")
}

func applyPublicBase(signed, publicBase string) string {
	publicBase = strings.TrimSpace(publicBase)
	if publicBase == "" {
		return signed
	}
	u, err := url.Parse(signed)
	if err != nil {
		return signed
	}
	if !strings.Contains(publicBase, "://") {
		publicBase = "https://" + publicBase
	}
	base, err := url.Parse(publicBase)
	if err != nil || base.Host == "" {
		return signed
	}
	u.Scheme = base.Scheme
	u.Host = base.Host
	if p := strings.TrimSuffix(base.Path, "/"); p != "" && !strings.HasSuffix(u.Path, p) {
		u.Path = p + u.Path
	}
	return u.String()
}
