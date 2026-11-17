//go:build windows

package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformCommandContextWrapsCmdShims(t *testing.T) {
	cmd := platformCommandContext(t.Context(), []string{`C:\Users\dev\AppData\Roaming\npm\codex.cmd`, "exec", "hello world"})
	if !strings.EqualFold(filepath.Base(cmd.Args[0]), "cmd.exe") {
		t.Fatalf("args=%q", cmd.Args)
	}
	if cmd.Args[len(cmd.Args)-1] != `"C:\Users\dev\AppData\Roaming\npm\codex.cmd" "exec" "hello world"` {
		t.Fatalf("cmd line=%q", cmd.Args[len(cmd.Args)-1])
	}
}

func TestPlatformCommandContextLeavesExeDirect(t *testing.T) {
	cmd := platformCommandContext(t.Context(), []string{`C:\tools\agentflow.exe`, "run"})
	if got := strings.Join(cmd.Args, " "); got != `C:\tools\agentflow.exe run` {
		t.Fatalf("args=%q", got)
	}
}

func TestQuoteForCmd(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", `""`},
		{"plain", "plain"},
		{"two words", `"two words"`},
		{`say "hi"`, `"say \"hi\""`},
	} {
		if got := quoteForCmd(tc.in); got != tc.want {
			t.Fatalf("quoteForCmd(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestLookupExecutableInDirUsesPathexExtensions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	if err := os.WriteFile(filepath.Join(dir, "codex.cmd"), []byte("@echo off\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, ok := lookupExecutableInDir(dir, "codex")
	if !ok || !strings.HasSuffix(strings.ToLower(path), "codex.cmd") {
		t.Fatalf("lookupExecutableInDir=%q %v", path, ok)
	}
	if _, ok := lookupExecutableInDir(dir, "missing"); ok {
		t.Fatal("missing binary resolved")
	}
}
