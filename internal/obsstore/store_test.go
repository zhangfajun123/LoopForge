package obsstore

import (
	"testing"
)

func TestConfigEnabledRequiresAllCredentials(t *testing.T) {
	full := Config{Endpoint: "obs.cn-north-4.myhuaweicloud.com", Bucket: "b", AccessKey: "ak", SecretKey: "sk"}
	if !full.Enabled() {
		t.Fatal("complete config should be enabled")
	}
	for _, cfg := range []Config{
		{},
		{Endpoint: full.Endpoint, Bucket: full.Bucket, AccessKey: full.AccessKey},
		{Endpoint: full.Endpoint, Bucket: full.Bucket, SecretKey: full.SecretKey},
	} {
		if cfg.Enabled() {
			t.Fatalf("incomplete config enabled: %+v", cfg)
		}
	}
}

func TestNewDisabledReturnsNilStore(t *testing.T) {
	store, err := New(Config{})
	if err != nil || store != nil {
		t.Fatalf("store=%v err=%v", store, err)
	}
}

func TestArtifactKeyIsStableAndSafe(t *testing.T) {
	if got := ArtifactKey("run/1", "../node", 2, "sub/../video.bin"); got != "runs/run-1/-node/2/sub/video.bin" {
		t.Fatalf("key=%q", got)
	}
	if got := SafeSegment(".."); got != "item" {
		t.Fatalf("segment=%q", got)
	}
	if got := normalizeEndpoint("https://obs.cn-north-4.myhuaweicloud.com/"); got != "obs.cn-north-4.myhuaweicloud.com" {
		t.Fatalf("endpoint=%q", got)
	}
}

func TestApplyPublicBaseKeepsSignature(t *testing.T) {
	signed := "https://bucket.obs.cn-north-4.myhuaweicloud.com/k?Signature=a&Expires=1"
	got := applyPublicBase(signed, "https://loopforge-dev.obs.cn-north-4.myhuaweicloud.com")
	want := "https://loopforge-dev.obs.cn-north-4.myhuaweicloud.com/k?Signature=a&Expires=1"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}
