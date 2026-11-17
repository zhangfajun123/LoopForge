//go:build !windows

package runner

import (
	"os"
	"path/filepath"
)

// platformExtraSearchPaths covers CLIs installed in user-local directories
// that a launchd/systemd environment does not add to PATH (the daemon often
// runs with a minimal /usr/bin:/bin PATH even though the user's shell finds
// ~/.local/bin, Homebrew, or nvm-installed CLIs).
func platformExtraSearchPaths(home string) []string {
	paths := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
	}
	// nvm installs Node-based CLIs (codex) under versioned directories.
	if entries, err := os.ReadDir(filepath.Join(home, ".nvm", "versions", "node")); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				paths = append(paths, filepath.Join(home, ".nvm", "versions", "node", entry.Name(), "bin"))
			}
		}
	}
	return paths
}
