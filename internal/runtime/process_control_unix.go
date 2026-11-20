//go:build !windows

package runtime

import (
	"errors"
	"os/exec"
	"syscall"
)

// ConfigureProcessTree gives a child its own process group so cancellation
// also stops shells and helper processes it started.
func ConfigureProcessTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return SignalProcessTree(cmd.Process.Pid, true)
	}
}

func ProcessTreeConfigured(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid
}

// SignalProcessTree sends TERM first, then KILL when force is true.
func SignalProcessTree(pid int, force bool) error {
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	err := syscall.Kill(-pid, sig)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// ProcessAlive reports whether the process group leader still exists.
func ProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
