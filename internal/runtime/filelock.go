package runtime

import "os"

// TryLockFile takes one non-blocking exclusive lock on f. A lock already held
// by another handle is reported as (false, nil); the platform implementation
// lives in filelock_unix.go / filelock_windows.go.
func TryLockFile(f *os.File) (bool, error) {
	return tryLockFile(f)
}

// UnlockFile releases a lock taken by TryLockFile.
func UnlockFile(f *os.File) error {
	return unlockFileHandle(f)
}
