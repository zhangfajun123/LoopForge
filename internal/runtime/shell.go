package runtime

// The runner executes two kinds of shell payloads: platform-owned scripts
// (workspace preparation, MCP provisioning) and user commands dispatched by
// the control plane (`sh -lc <script>`). POSIX shells do not exist on Windows,
// so every execution path goes through this adapter instead of hard-coding
// `sh -lc`. On Unix it keeps the historical behaviour byte for byte.

// ShellCommand returns the platform shell invocation that runs script.
func ShellCommand(script string) []string {
	return shellCommand(script)
}

// ShellQuote encodes value so the platform shell sees it as one literal word.
func ShellQuote(value string) string {
	return shellQuote(value)
}

// NormalizeShellCommand rewrites a POSIX shell argv the control plane
// dispatched into the platform equivalent. On Unix the argv is returned
// unchanged, so runner builds that predate this adapter are unaffected.
func NormalizeShellCommand(argv []string) []string {
	return normalizeShellCommand(argv)
}

// privateHomeEnvironment maps a private HOME root onto the variables each
// platform's CLIs actually read, so an Agent CLI cannot fall back to the
// operator's real profile directory.
func privateHomeEnvironment(root string) map[string]string {
	return privateHomeEnv(root)
}
