//go:build windows

package runtime

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFileRegion locks the whole file: length 0xFFFFFFFF/0xFFFFFFFF means
// "all bytes, present and future".
const (
	lockFileMaxLow  = 0xFFFFFFFF
	lockFileMaxHigh = 0xFFFFFFFF
)

// tryLockFile attempts one non-blocking exclusive LockFileEx. Windows locks
// belong to the handle, so a second handle in the same process conflicts just
// like another process would: workspace exclusivity holds without an extra
// in-process mutex.
func tryLockFile(f *os.File) (bool, error) {
	overlapped := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, lockFileMaxLow, lockFileMaxHigh, overlapped)
	if err == nil {
		return true, nil
	}
	if err == windows.ERROR_LOCK_VIOLATION || err == windows.ERROR_IO_PENDING {
		return false, nil
	}
	return false, err
}

func unlockFileHandle(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockFileMaxLow, lockFileMaxHigh, new(windows.Overlapped))
}
