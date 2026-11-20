//go:build windows

package main

import (
	"os"
	"os/exec"
)

func runnerSignals() []os.Signal { return []os.Signal{os.Interrupt} }

func restartUpdatedRunner(binary string) error {
	cmd := exec.Command(binary, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Start()
}
