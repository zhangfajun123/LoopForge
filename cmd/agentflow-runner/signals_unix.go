//go:build !windows

package main

import (
	"os"
	"syscall"
)

func runnerSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }

func restartUpdatedRunner(binary string) error {
	return syscall.Exec(binary, os.Args, os.Environ())
}
