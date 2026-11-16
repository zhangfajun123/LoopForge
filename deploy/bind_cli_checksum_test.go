package deploy

import (
	"strings"
	"testing"
)

func bindCLIScriptText(t *testing.T) string {
	t.Helper()
	script := string(BindCLIScript())
	if script == "" {
		t.Fatal("installer content is empty")
	}
	return script
}

func TestBindCLIScriptVerifiesDownloadedBinaries(t *testing.T) {
	script := bindCLIScriptText(t)
	for _, want := range []string{
		"--sha256) EXPECTED_SHA256=",
		"#   --sha256 <hex>",
		"verify_binary_sha256 \"$TEMP_BINARY\"",
		`remote_sha256 "${SERVER}/runners/agentflow-runner?os=${OS}&arch=${ARCH}&metadata=1"`,
		`remote_sha256 "${SERVER}/runners/agentflow-gate-mcp`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("installer misses %q", want)
		}
	}
	// The runner must be checked before it is moved into place, and the gate MCP
	// helper before it is marked executable.
	verify := strings.Index(script, `verify_binary_sha256 "$TEMP_BINARY"`)
	move := strings.Index(script, `mv -f "$TEMP_BINARY" "$BINARY"`)
	if verify < 0 || move < 0 || verify > move {
		t.Fatalf("runner checksum check must run before install (verify=%d move=%d)", verify, move)
	}
}

func TestBindCLIScriptHasAFailsClosedMismatch(t *testing.T) {
	script := bindCLIScriptText(t)
	if !strings.Contains(script, "sha256 校验失败") {
		t.Fatal("a checksum mismatch must abort the install")
	}
	// Missing metadata is a warning, not a hard failure: platforms that do not
	// expose ?metadata=1 must stay installable.
	if !strings.Contains(script, "已跳过完整性校验") {
		t.Fatal("a platform without metadata must degrade to a warning")
	}
}

func TestBindCLIScriptStaysBash3Compatible(t *testing.T) {
	// macOS ships bash 3.2: ${var,,} and similar case operators abort the script
	// before anything is installed.
	script := bindCLIScriptText(t)
	for _, forbidden := range []string{"${actual,,}", "${expected,,}", "${SERVER,,}"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("installer uses bash-4-only expansion %q", forbidden)
		}
	}
	for _, tool := range []string{"sha256sum", "shasum -a 256", "openssl dgst -sha256"} {
		if !strings.Contains(script, tool) {
			t.Errorf("sha256 helper misses the %q fallback", tool)
		}
	}
}