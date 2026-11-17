//go:build !windows

package runtime

import (
	"strings"

	"agentflow/internal/domain"
)

func shellCommand(script string) []string {
	return []string{"sh", "-lc", script}
}

// shellQuote wraps value in single quotes, escaping embedded quotes with the
// POSIX '"'"' sequence.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// normalizeShellCommand is a no-op on Unix: the control plane already emits
// compatible `sh -c` / `sh -lc` payloads.
func normalizeShellCommand(argv []string) []string {
	return argv
}

func privateHomeEnv(root string) map[string]string {
	return map[string]string{"HOME": root}
}

// preparationScript wraps an environment-preparation node with the variables
// and repository checkout the script expects. Unix keeps its historical
// `export` / `cd --` preamble byte for byte.
func preparationScript(script, scope string, repo *domain.RepositoryContext) string {
	prefix := "export AGENTFLOW_WORKSPACE=\"$PWD\"\nexport AGENTFLOW_EXECUTION_SCOPE=" + shellQuote(scope) + "\n"
	if repo != nil {
		prefix += "export AGENTFLOW_REPOSITORY_ID=" + shellQuote(repo.RepositoryID) + "\nexport AGENTFLOW_REPOSITORY_NAME=" + shellQuote(repo.Name) + "\nexport AGENTFLOW_REPOSITORY_DIR=" + shellQuote(repo.WorkspaceDir) + "\ncd -- " + shellQuote(repo.WorkspaceDir) + " || exit $?\n"
	}
	return prefix + script
}

// pythonInlineScript runs a python preparation script with the POSIX python3
// entry point.
func pythonInlineScript(script string) string {
	return "python3 -c " + shellQuote(script)
}
