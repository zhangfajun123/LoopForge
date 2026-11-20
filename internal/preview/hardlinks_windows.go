//go:build windows

package preview

import (
	"os"

	"golang.org/x/sys/windows"
)

func fileHasMultipleLinks(path string, _ os.FileInfo) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return false
	}
	return info.NumberOfLinks > 1
}
