//go:build windows

package runtime

import (
	"os"

	"golang.org/x/sys/windows"
)

// workspaceFileIdentity exposes the volume serial number and file index that
// identify a workspace directory on this host. Windows has no device/inode
// pair; the volume serial plus the 64-bit file index is the stable equivalent.
func workspaceFileIdentity(path string, _ os.FileInfo) (device, inode uint64, ok bool) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return 0, 0, false
	}
	return uint64(info.VolumeSerialNumber), uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow), true
}
