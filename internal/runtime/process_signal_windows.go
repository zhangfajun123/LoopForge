//go:build windows

package runtime

import "os"

func processExitSignal(*os.ProcessState) string { return "" }
func argumentTooLarge(error) bool               { return false }
