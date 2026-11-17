package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// The probe must read the flag token, not any occurrence of the word: opencode
// 2.x documents "provider/model#variant" in a help line that must not be taken
// as having --variant.
func TestMentionsFlag(t *testing.T) {
	modern := `  -m, --model        model to use in the format of provider/model#variant`
	if mentionsFlag(modern, "--variant") {
		t.Fatal("provider/model#variant help line counted as a --variant flag")
	}
	legacy := `      --variant      model variant (provider-specific reasoning effort, e.g., high, max, minimal)`
	if !mentionsFlag(legacy, "--variant") {
		t.Fatal("legacy help line did not register --variant")
	}
	if !mentionsFlag("  --variant=high   inline", "--variant") {
		t.Fatal("inline form not detected")
	}
}

// helpListsFlag runs the real binary's help; a script that prints the flag is
// enough to prove the plumbing without depending on an installed CLI.
func TestHelpListsFlagProbesBinary(t *testing.T) {
	dir := t.TempDir()
	withFlag := filepath.Join(dir, "with-flag")
	if err := os.WriteFile(withFlag, []byte("#!/bin/sh\necho '      --variant   model variant'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	without := filepath.Join(dir, "without-flag")
	if err := os.WriteFile(without, []byte("#!/bin/sh\necho '  -m, --model   provider/model#variant'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !helpListsFlag(withFlag, "run", "--variant") {
		t.Fatal("probe missed a binary that lists --variant")
	}
	if helpListsFlag(without, "run", "--variant") {
		t.Fatal("probe accepted a binary that only mentions provider/model#variant")
	}
	if helpListsFlag(filepath.Join(dir, "missing"), "run", "--variant") {
		t.Fatal("probe reported support for a missing binary")
	}
}