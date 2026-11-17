package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentflow/internal/domain"
)

// A provider task on a machine without a sandbox used to inherit the runner's
// real ~/.codex, so the host's config.toml leaked into the task and could fail
// it. The private HOME must exist, and in local credential mode the CLI must
// stay logged in without bringing its configuration along.
func TestBindProviderPrivateHomeIsolatesHostConfigAndSeedsLogin(t *testing.T) {
	realHome := t.TempDir()
	t.Setenv("HOME", realHome)
	writeFile(t, filepath.Join(realHome, ".codex", "auth.json"), `{"token":"login"}`)
	writeFile(t, filepath.Join(realHome, ".codex", "config.toml"), `model_provider = "agentflow"`)

	workspace := t.TempDir()
	env, err := bindProviderPrivateHome(workspace, "", "codex", domain.RuntimeAuthLocal, nil)
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	privateHome := filepath.Join(workspace, ".agentflow", "private-home", "codex")
	if env["HOME"] != privateHome {
		t.Fatalf("HOME = %q, want %q", env["HOME"], privateHome)
	}
	if env["CODEX_HOME"] != filepath.ToSlash(filepath.Join(privateHome, ".codex")) {
		t.Fatalf("CODEX_HOME = %q", env["CODEX_HOME"])
	}
	if !strings.HasPrefix(env["TMPDIR"], filepath.Join(workspace, ".agentflow", "tmp")) {
		t.Fatalf("TMPDIR = %q", env["TMPDIR"])
	}
	if _, err := os.Stat(filepath.Join(privateHome, ".codex", "auth.json")); err != nil {
		t.Fatalf("login state was not seeded: %v", err)
	}
	if _, err := os.Stat(filepath.Join(privateHome, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("host config.toml leaked into the private HOME (err=%v)", err)
	}
	info, err := os.Stat(filepath.Join(privateHome, ".codex", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("seeded login perms = %v, want 0600", info.Mode().Perm())
	}
}

// With personal credentials the key is injected, so no login state is copied.
func TestBindProviderPrivateHomeSkipsLoginForPersonalCredentials(t *testing.T) {
	realHome := t.TempDir()
	t.Setenv("HOME", realHome)
	writeFile(t, filepath.Join(realHome, ".codex", "auth.json"), `{"token":"login"}`)

	workspace := t.TempDir()
	env, err := bindProviderPrivateHome(workspace, "", "codex", domain.RuntimeAuthPersonal, nil)
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".agentflow", "private-home", "codex", ".codex", "auth.json")); !os.IsNotExist(err) {
		t.Fatalf("login state was copied for personal credentials (err=%v)", err)
	}
	if env["HOME"] == "" {
		t.Fatal("HOME was not isolated")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}