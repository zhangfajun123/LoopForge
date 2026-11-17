//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// createNewProcessGroup is CREATE_NEW_PROCESS_GROUP; CTRL_BREAK can then reach
// the new daemon without hitting this process.
const createNewProcessGroup = 0x00000200

// restartSignals are the termination signals the daemon handles gracefully.
// Windows delivers console control events as Interrupt; SIGTERM does not exist.
func restartSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

// restartRunner starts the updated binary as a detached child and lets the
// current process exit. Windows cannot exec in place, and a running image
// cannot be replaced, so the update path stages the new binary for the next
// start instead (see internal/runner/update.go).
func restartRunner(binary string) error {
	cmd := exec.Command(binary, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start updated runner: %w", err)
	}
	log.Printf("started updated runner pid=%d", cmd.Process.Pid)
	return nil
}
