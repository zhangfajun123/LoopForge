//go:build windows

package runtime

import (
	"reflect"
	"strings"
	"testing"
)

func TestShellCommandUsesPowerShell(t *testing.T) {
	got := ShellCommand("Get-Location")
	if len(got) < 2 || !strings.EqualFold(got[0], "powershell.exe") || got[len(got)-2] != "-Command" || got[len(got)-1] != "Get-Location" {
		t.Fatalf("ShellCommand=%q", got)
	}
}

func TestShellQuoteDoublesSingleQuotes(t *testing.T) {
	if got, want := ShellQuote("it's"), "'it''s'"; got != want {
		t.Fatalf("ShellQuote=%q want %q", got, want)
	}
}

func TestNormalizeShellCommandRewritesPosixShellEntry(t *testing.T) {
	for _, argv := range [][]string{
		{"sh", "-lc", "echo hi"},
		{"/bin/sh", "-c", "echo hi"},
		{"bash", "-l", "-c", "echo hi"},
		{`C:\Program Files\Git\bin\bash.exe`, "-lc", "echo hi"},
	} {
		got := NormalizeShellCommand(argv)
		if len(got) == 0 || !strings.EqualFold(got[0], "powershell.exe") {
			t.Fatalf("NormalizeShellCommand(%q)=%q", argv, got)
		}
		if got[len(got)-1] != "echo hi" {
			t.Fatalf("NormalizeShellCommand(%q)=%q", argv, got)
		}
	}
}

func TestNormalizeShellCommandLeavesNativeArgv(t *testing.T) {
	want := []string{`C:\tools\codex.exe`, "exec", "prompt"}
	if got := NormalizeShellCommand(want); !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeShellCommand=%q want %q", got, want)
	}
}
