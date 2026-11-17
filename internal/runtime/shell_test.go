//go:build !windows

package runtime

import (
	"reflect"
	"testing"

	"agentflow/internal/domain"
)

func TestShellCommandUsesPosixLoginShell(t *testing.T) {
	script := "printf '%s\\n' \"$PWD\""
	if got, want := ShellCommand(script), []string{"sh", "-lc", script}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ShellCommand=%q want %q", got, want)
	}
}

func TestShellQuoteEscapesSingleQuotesForPosix(t *testing.T) {
	if got, want := ShellQuote("it's a '$HOME'"), `'it'"'"'s a '"'"'$HOME'"'"''`; got != want {
		t.Fatalf("ShellQuote=%q want %q", got, want)
	}
}

func TestNormalizeShellCommandKeepsPosixArgv(t *testing.T) {
	argv := []string{"bash", "-lc", "echo hi"}
	if got := NormalizeShellCommand(argv); !reflect.DeepEqual(got, argv) {
		t.Fatalf("NormalizeShellCommand=%q want %q", got, argv)
	}
}

func TestPreparationScriptKeepsPosixPreamble(t *testing.T) {
	repo := &domain.RepositoryContext{RepositoryID: "r1", Name: "demo", WorkspaceDir: "repos/demo"}
	got := preparationScript("echo ok", "repository", repo)
	want := "export AGENTFLOW_WORKSPACE=\"$PWD\"\nexport AGENTFLOW_EXECUTION_SCOPE='repository'\n" +
		"export AGENTFLOW_REPOSITORY_ID='r1'\nexport AGENTFLOW_REPOSITORY_NAME='demo'\nexport AGENTFLOW_REPOSITORY_DIR='repos/demo'\ncd -- 'repos/demo' || exit $?\necho ok"
	if got != want {
		t.Fatalf("preparationScript=%q want %q", got, want)
	}
}
