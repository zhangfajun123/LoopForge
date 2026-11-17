//go:build !windows

package runner

import "os"

// installUpdatedExecutable replaces the executable in place. Unix allows
// renaming over a running image, and the entry point re-execs the new file.
func installUpdatedExecutable(tmp, target string) (installed bool, err error) {
	if err := os.Rename(tmp, target); err != nil {
		return false, err
	}
	return true, nil
}
