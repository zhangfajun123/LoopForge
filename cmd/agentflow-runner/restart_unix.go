//go:build !windows

package main

import (
	"os"
	"syscall"
)

// restartSignals are the termination signals the daemon handles gracefully.
func restartSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// restartRunner replaces the current process image with the updated binary,
// keeping the same argument list, environment and exit status.
func restartRunner(binary string) error {
	return syscall.Exec(binary, os.Args, os.Environ())
}
