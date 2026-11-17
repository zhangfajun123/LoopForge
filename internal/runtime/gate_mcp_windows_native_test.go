//go:build windows

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeGateMCPNamesWindowsCommandWithExe(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "agentflow-gate-mcp-windows-amd64.exe")
	if err := os.WriteFile(source, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTFLOW_GATE_MCP_BINARY", source)
	workspace := filepath.Join(dir, "workspace")
	configRoot := filepath.Join(workspace, ".agentflow", "config", "reviewer")
	if err := os.MkdirAll(configRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	server, err := materializeGateMCP(workspace, configRoot, ".agentflow/config/reviewer/gate-mcp", ".agentflow/gate-results/gate.json", "local")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(server.Command, "gate-mcp.exe") {
		t.Fatalf("command=%q", server.Command)
	}
	if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(server.Command))); err != nil {
		t.Fatalf("materialized gate binary: %v", err)
	}
}
