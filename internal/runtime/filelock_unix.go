//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

// tryLockFile attempts one non-blocking exclusive flock. A busy lock is
// reported as (false, nil); callers poll with a context deadline.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return false, nil
	}
	return false, err
}

func unlockFileHandle(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
