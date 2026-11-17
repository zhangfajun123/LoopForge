package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeFakeOpenCode writes a script that answers `--help`, `run --help` and
// `--version` the way a given opencode release does, so the decision can be
// tested against both releases without either installed.
func writeFakeOpenCode(t *testing.T, dir, name, helpBody, version string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"--version) echo '" + version + "' ;;\n" +
		"*) cat <<'HELP'\n" + helpBody + "\nHELP\n ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

const opencodeV1Help = `USAGE
  opencode run [flags] [<message...>]
FLAGS
  --format choice         Output format (choices: default, json)
  --thinking              Show thinking blocks
  --variant               model variant (provider-specific reasoning effort)
  -m, --model string      Model to use`

const opencodeV2Help = `USAGE
  opencode run [flags] [<message...>]
FLAGS
  --format choice         Output format (choices: default, json)
  --thinking              Show thinking blocks
  --standalone            Run with a private server instead of the background service
  -m, --model string      Model to use in the format provider/model#variant`

// decideAgentCLIArgv is the code the runner executes for every opencode task.
// Both release shapes are covered here, so a regression shows up as a failing
// test instead of another failed run.
func TestDecideAgentCLIArgv(t *testing.T) {
	dir := t.TempDir()
	v1 := writeFakeOpenCode(t, dir, "opencode-1", opencodeV1Help, "1.18.25")
	v2 := writeFakeOpenCode(t, dir, "opencode-2", opencodeV2Help, "2.2.5")
	d := &Daemon{}
	base := func(bin string) []string {
		return []string{bin, "run", "--format", "json", "--thinking", "-m", "agentflow/deepseek-v4.1-flash-1017", "--variant", "high", "task prompt"}
	}

	t.Run("2.x folds the effort and takes a private server", func(t *testing.T) {
		argv, decision := d.decideAgentCLIArgv("opencode", false, "", base(v2))
		want := []string{v2, "run", "--standalone", "--format", "json", "--thinking", "-m", "agentflow/deepseek-v4.1-flash-1017#high", "task prompt"}
		if !reflect.DeepEqual(argv, want) {
			t.Fatalf("argv = %v\nwant %v", argv, want)
		}
		if !decision.LocalBinary || !decision.Probed || !decision.StandaloneSupported || decision.VariantSupported {
			t.Fatalf("decision = %+v", decision)
		}
	})

	t.Run("1.x is left alone", func(t *testing.T) {
		argv, decision := d.decideAgentCLIArgv("opencode", false, "", base(v1))
		if !reflect.DeepEqual(argv, base(v1)) {
			t.Fatalf("argv = %v\nwant %v", argv, base(v1))
		}
		if !decision.VariantSupported || decision.StandaloneSupported {
			t.Fatalf("decision = %+v", decision)
		}
	})

	t.Run("a binary this host does not have is not probed", func(t *testing.T) {
		missing := filepath.Join(dir, "opencode-missing")
		argv, decision := d.decideAgentCLIArgv("opencode", true, "docker", base(missing))
		if !reflect.DeepEqual(argv, base(missing)) {
			t.Fatalf("argv = %v\nwant %v", argv, base(missing))
		}
		if decision.LocalBinary || decision.Probed {
			t.Fatalf("decision = %+v", decision)
		}
	})

	t.Run("other providers are untouched", func(t *testing.T) {
		argv, decision := d.decideAgentCLIArgv("codex", false, "", base(v2))
		if !reflect.DeepEqual(argv, base(v2)) {
			t.Fatalf("argv = %v\nwant %v", argv, base(v2))
		}
		if decision.Probed || decision.LocalBinary {
			t.Fatalf("decision = %+v", decision)
		}
	})
}