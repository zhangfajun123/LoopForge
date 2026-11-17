//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

// fileOwner returns the UID that created a regular file. Git creates the
// index lock with O_EXCL, so the owner is the process that holds it.
func fileOwner(info os.FileInfo) (uint32, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return stat.Uid, true
}

func currentProcessOwner() uint32 {
	return uint32(os.Geteuid())
}
