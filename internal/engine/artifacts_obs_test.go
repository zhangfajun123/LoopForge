package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agentflow/internal/domain"
	"agentflow/internal/obsstore"
)

type fakeBlobStore struct {
	objects map[string][]byte
}

func (f *fakeBlobStore) Enabled() bool { return true }

func (f *fakeBlobStore) Put(_ context.Context, key string, src io.Reader, size int64) error {
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	data, err := io.ReadAll(io.LimitReader(src, size+1))
	if err != nil {
		return err
	}
	if int64(len(data)) != size {
		return fmt.Errorf("size mismatch: got %d want %d", len(data), size)
	}
	f.objects[key] = data
	return nil
}

func (f *fakeBlobStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := f.objects[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeBlobStore) Delete(_ context.Context, key string) error {
	delete(f.objects, key)
	return nil
}

func (f *fakeBlobStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "https://obs.example/object", nil
}

func TestCollectArtifactsStoresOversizedBodyInOBS(t *testing.T) {
	workspace := t.TempDir()
	body := bytes.Repeat([]byte("a"), maxArtifactBytes+1024)
	if err := os.WriteFile(filepath.Join(workspace, "video.bin"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	node := domain.WorkflowNode{ID: "node", ArtifactOutputs: []domain.ArtifactContract{{Name: "video", Path: "video.bin", MediaType: "application/octet-stream", Required: true}}}
	run := domain.Run{ID: "run-obs", Nodes: map[string]*domain.NodeRun{"node": {Attempt: 2}}}
	blobs := &fakeBlobStore{}

	artifacts, err := collectArtifactsWithBlobs(workspace, run, node, "output", blobs)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts=%v", artifacts)
	}
	art := artifacts[0]
	if art.StorageBackend != "obs" || art.StorageKey == "" {
		t.Fatalf("storage backend=%q key=%q", art.StorageBackend, art.StorageKey)
	}
	if art.Content != "" {
		t.Fatal("oversized artifact must not be inlined")
	}
	sum := sha256.Sum256(body)
	if art.SHA256 != fmt.Sprintf("%x", sum) || art.Size != int64(len(body)) {
		t.Fatalf("sha=%q size=%d", art.SHA256, art.Size)
	}
	if !bytes.Equal(blobs.objects[art.StorageKey], body) {
		t.Fatal("object body mismatch")
	}
}

func TestCollectArtifactsKeepsSmallBodyInline(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "notes.md"), []byte("small"), 0o644); err != nil {
		t.Fatal(err)
	}
	node := domain.WorkflowNode{ID: "node", ArtifactOutputs: []domain.ArtifactContract{{Name: "notes", Path: "notes.md", Required: true}}}
	run := domain.Run{ID: "run-small", Nodes: map[string]*domain.NodeRun{"node": {Attempt: 1}}}

	artifacts, err := collectArtifactsWithBlobs(workspace, run, node, "output", &fakeBlobStore{})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].StorageBackend != "db" || artifacts[0].Content != "small" {
		t.Fatalf("artifacts=%+v", artifacts)
	}
}

func TestMaterializeInputsDownloadsOBSArtifact(t *testing.T) {
	workspace := t.TempDir()
	body := []byte("remote body")
	sum := sha256.Sum256(body)
	blobs := &fakeBlobStore{objects: map[string][]byte{"runs/r/n/1/source/report.md": body}}
	run := domain.Run{Artifacts: []domain.Artifact{{
		NodeID: "source", Name: "report", MediaType: "text/markdown", Size: int64(len(body)),
		SHA256: fmt.Sprintf("%x", sum), StorageBackend: "obs", StorageKey: "runs/r/n/1/source/report.md",
	}}}
	wf := domain.WorkflowDefinition{Edges: []domain.WorkflowEdge{{Source: "source", Target: "target"}}}
	node := domain.WorkflowNode{ID: "target"}

	if _, err := materializeInputsWithBlobs(workspace, wf, node, run, blobs); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(workspace, ".agentflow", "inputs", "source", "report.md"))
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("body=%q err=%v", got, err)
	}
}

func TestCollectArtifactsUsesRunnerUploadMetadata(t *testing.T) {
	node := domain.WorkflowNode{ID: "node", ArtifactOutputs: []domain.ArtifactContract{{Name: "big", Path: "big.bin", Required: true}}}
	run := domain.Run{ID: "run-upload", Nodes: map[string]*domain.NodeRun{"node": {Attempt: 1}}}
	reported := newReportedOutputs(nil)
	reported.absorb(&domain.RunnerTaskOutputs{
		TooLarge: []string{"big.bin"},
		Uploads:  map[string]domain.ArtifactUpload{"big.bin": {StorageKey: "k/big.bin", SHA256: "abc", Size: 2 << 20}},
	})
	if err := validateArtifactOutputsAllow(reported, run, node, true); err != nil {
		t.Fatal(err)
	}
	artifacts, err := collectArtifactsModeWithBlobs(reported, run, node, "output", false, &fakeBlobStore{})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].StorageKey != "k/big.bin" || artifacts[0].Content != "" || artifacts[0].Size != 2<<20 {
		t.Fatalf("artifacts=%+v", artifacts)
	}
}

var _ obsstore.BlobStore = (*fakeBlobStore)(nil)
