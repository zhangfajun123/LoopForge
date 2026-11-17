//go:build windows

package preview

import "os"

// hasMultipleHardlinks reports whether info has more than one directory entry.
// Windows exposes the hardlink count only through GetFileInformationByHandle,
// which os.FileInfo does not carry; preview archiving keeps working without
// the check on that platform.
func hasMultipleHardlinks(os.FileInfo) bool {
	return false
}
