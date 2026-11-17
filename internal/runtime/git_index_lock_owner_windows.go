//go:build windows

package runtime

import "os"

// fileOwner reports the owner of a regular file. Windows has no UID on
// os.FileInfo; index-lock inspection by owner is only used on Linux, where
// /proc exposes process descriptors.
func fileOwner(os.FileInfo) (uint32, bool) {
	return 0, false
}

func currentProcessOwner() uint32 {
	return 0
}
