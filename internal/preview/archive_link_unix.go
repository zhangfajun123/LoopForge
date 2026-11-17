//go:build !windows

package preview

import (
	"os"
	"syscall"
)

// hasMultipleHardlinks reports whether info has more than one directory entry.
// Preview archives reject hardlinks because unpacking them would duplicate
// content that a single member cannot express.
func hasMultipleHardlinks(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
