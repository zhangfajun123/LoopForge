//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

// workspaceFileIdentity exposes the device and inode that identify a workspace
// directory on this host, so preparation stamps are invalidated when a
// different filesystem backs the same path.
func workspaceFileIdentity(_ string, info os.FileInfo) (device, inode uint64, ok bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return uint64(stat.Dev), stat.Ino, true
}
