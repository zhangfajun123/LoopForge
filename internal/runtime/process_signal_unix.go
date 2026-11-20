//go:build !windows

package runtime

import (
	"errors"
	"os"
	"syscall"
)

func processExitSignal(state *os.ProcessState) string {
	status, ok := state.Sys().(interface {
		Signaled() bool
		Signal() syscall.Signal
	})
	if !ok || !status.Signaled() {
		return ""
	}
	return status.Signal().String()
}

func argumentTooLarge(err error) bool { return errors.Is(err, syscall.E2BIG) }
