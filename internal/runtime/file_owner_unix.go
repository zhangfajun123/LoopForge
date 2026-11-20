//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

func fileOwnerID(info os.FileInfo) (uint32, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint32(stat.Uid), true
}

func currentUserID() uint32 { return uint32(os.Geteuid()) }
func isRootUser() bool      { return os.Geteuid() == 0 }
func dockerHostIdentity() (int, int, bool) {
	uid, gid := os.Geteuid(), os.Getegid()
	return uid, gid, uid != 0
}
