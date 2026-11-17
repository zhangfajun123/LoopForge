//go:build windows

package runtime

import (
	"os/exec"
	"path/filepath"
	"strings"

	"agentflow/internal/domain"
)

// powershellArgs runs one script non-interactively. -NoProfile keeps the
// operator's PowerShell profile out of task execution, and Bypass avoids
// installation-conditional ExecutionPolicy failures on a service account.
var powershellArgs = []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command"}

func shellCommand(script string) []string {
	return append(append([]string{}, powershellArgs...), script)
}

// shellQuote wraps value in a PowerShell single-quoted string, doubling
// embedded single quotes (PowerShell has no backslash escape).
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// normalizeShellCommand rewrites the POSIX shells the control plane emits
// (`sh -lc <script>`, `bash -c <script>`, `/bin/sh -c <script>`) into a
// PowerShell invocation. Only the entry point is translated: the script body
// keeps its original syntax and must already be PowerShell.
//
// A `python3`/`python` entry point is resolved to an interpreter that actually
// exists on this machine; argv elements are passed as separate arguments, so
// no additional quoting is required.
func normalizeShellCommand(argv []string) []string {
	if len(argv) == 0 {
		return argv
	}
	first := strings.ToLower(strings.TrimSpace(argv[0]))
	if isPosixShell(first) {
		script, ok := shellScriptArgument(argv)
		if !ok {
			return argv
		}
		return shellCommand(script)
	}
	if resolved, ok := resolvePythonInterpreter(argv); ok {
		return resolved
	}
	return argv
}

func isPosixShell(name string) bool {
	name = strings.TrimSuffix(name, ".exe")
	switch name {
	case "sh", "bash", "dash", "zsh", "ksh", "ash":
		return true
	}
	// /bin/sh, /usr/bin/bash, C:\Program Files\Git\bin\bash.exe
	base := baseName(name)
	return base != "" && base != name && isPosixShell(base)
}

func baseName(path string) string {
	if index := strings.LastIndexAny(path, "/\\"); index >= 0 {
		return path[index+1:]
	}
	return path
}

// shellScriptArgument returns the script text of `sh -c script` and
// `sh -lc script`. Login flags may be combined (`-lc`) or separate (`-l -c`).
func shellScriptArgument(argv []string) (string, bool) {
	for index := 1; index < len(argv); index++ {
		flag := argv[index]
		if !strings.HasPrefix(flag, "-") || flag == "-" {
			return "", false
		}
		if strings.Contains(flag, "c") {
			if index+1 >= len(argv) {
				return "", false
			}
			return argv[index+1], true
		}
	}
	return "", false
}

// resolvePythonInterpreter maps `python3 -c <script>` onto an installed
// interpreter. Windows installs the launcher as `python`/`py`, and scripts
// keep working because exec passes each argv element as one argument.
func resolvePythonInterpreter(argv []string) ([]string, bool) {
	first := strings.ToLower(strings.TrimSuffix(baseName(argv[0]), ".exe"))
	if first != "python3" && first != "python" {
		return nil, false
	}
	if _, err := exec.LookPath(argv[0]); err == nil {
		return argv, true
	}
	for _, candidate := range []string{"python", "python3", "py"} {
		if path, err := exec.LookPath(candidate); err == nil {
			resolved := append([]string{path}, argv[1:]...)
			return resolved, true
		}
	}
	return nil, false
}

func privateHomeEnv(root string) map[string]string {
	return map[string]string{
		"HOME":        root,
		"USERPROFILE": root,
		"TEMP":        root,
		"TMP":         root,
	}
}

// preparationScript wraps an environment-preparation node with the variables
// and repository checkout the script expects, using PowerShell syntax. The
// node body itself is PowerShell on Windows.
func preparationScript(script, scope string, repo *domain.RepositoryContext) string {
	var b strings.Builder
	b.WriteString("$env:AGENTFLOW_WORKSPACE = (Get-Location).Path\n")
	b.WriteString("$env:AGENTFLOW_EXECUTION_SCOPE = " + shellQuote(scope) + "\n")
	if repo != nil {
		b.WriteString("$env:AGENTFLOW_REPOSITORY_ID = " + shellQuote(repo.RepositoryID) + "\n")
		b.WriteString("$env:AGENTFLOW_REPOSITORY_NAME = " + shellQuote(repo.Name) + "\n")
		b.WriteString("$env:AGENTFLOW_REPOSITORY_DIR = " + shellQuote(repo.WorkspaceDir) + "\n")
		b.WriteString("Set-Location -LiteralPath " + shellQuote(filepath.FromSlash(repo.WorkspaceDir)) + "\n")
	}
	b.WriteString(script)
	return b.String()
}

// pythonInlineScript runs a python preparation script through the Windows
// `python` launcher; python3 is rarely installed under that name.
func pythonInlineScript(script string) string {
	return "python -c " + shellQuote(script)
}
