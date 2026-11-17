package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// Windows bundles name the executable with an .exe suffix; the resolver
// accepts a deployment path with or without the suffix.
func TestResolveGateMCPBinaryWindowsExtension(t *testing.T) {
	dir := t.TempDir()
	pattern := filepath.Join(dir, "agentflow-gate-mcp-{os}-{arch}")
	t.Setenv("AGENTFLOW_GATE_MCP_BINARY", pattern)
	want := filepath.Join(dir, "agentflow-gate-mcp-windows-amd64.exe")
	if err := os.WriteFile(want, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := resolveGateMCPBinary("windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolved=%q want %q", got, want)
	}
}
