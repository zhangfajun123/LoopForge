//go:build windows

package runner

import "os"

// pendingUpdateSuffix marks a verified download that the service wrapper swaps
// in on the next start. A running .exe cannot be replaced on Windows.
const pendingUpdateSuffix = ".pending"

// installUpdatedExecutable stages the new image instead of replacing the
// running executable, and reports installed=false so the daemon keeps serving
// until the next restart applies the update.
func installUpdatedExecutable(tmp, target string) (installed bool, err error) {
	pending := target + pendingUpdateSuffix
	// A previously staged image is no longer running, so it can be replaced.
	if err := os.Remove(pending); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.Rename(tmp, pending); err != nil {
		return false, err
	}
	return false, nil
}
