//go:build !windows

package runtime

import "strings"

func mcpCallHelperName() string {
	return "agentflow-helper"
}

func mcpCallWrapperName() string {
	return "agentflow"
}

// mcpCallWrapperScript is the workspace `agentflow` shim a script node calls
// for `agentflow mcp call`; it prefers the copied helper and falls back to the
// installed runner binary.
func mcpCallWrapperScript(helperPath, fallback string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	if strings.TrimSpace(helperPath) != "" {
		b.WriteString("if [ -x " + shellSingleQuote(helperPath) + " ]; then\n")
		b.WriteString("  exec " + shellSingleQuote(helperPath) + " \"$@\"\n")
		b.WriteString("fi\n")
	}
	if strings.TrimSpace(fallback) != "" {
		b.WriteString("if [ -x " + shellSingleQuote(fallback) + " ]; then\n")
		b.WriteString("  exec " + shellSingleQuote(fallback) + " \"$@\"\n")
		b.WriteString("fi\n")
	}
	b.WriteString("echo \"agentflow: mcp helper is not available in this sandbox\" >&2\n")
	b.WriteString("exit 127\n")
	return b.String()
}

// mcpCallPathOverlay is a no-op on Unix: the shell-function preamble already
// makes `agentflow` resolvable without touching the user's PATH.
func mcpCallPathOverlay(map[string]string, string) {}
