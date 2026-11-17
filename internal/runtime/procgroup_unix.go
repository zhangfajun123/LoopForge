//go:build !windows

package runtime

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup isolates a command in its own process group so a
// cancel can reach the CLI and every descendant it spawned.
func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// configureCommandCancel mirrors configureProcessGroup for exec.CommandContext:
// cancelling the context kills the whole group instead of only the leader.
func configureCommandCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// killProcessGroup terminates the group of a started command. force escalates
// from a polite signal to SIGKILL.
func killProcessGroup(cmd *exec.Cmd, force bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	_ = syscall.Kill(-cmd.Process.Pid, signal)
}

// killProcessTree force-terminates the process group led by pid.
func killProcessTree(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// processAlive reports whether pid belongs to a live process.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// exitStatusSignal reports the terminating signal name for a process state
// that was signalled, and whether the platform records one.
func exitStatusSignal(state interface{ Sys() any }) (string, bool) {
	status, ok := state.Sys().(interface {
		Signaled() bool
		Signal() syscall.Signal
	})
	if !ok || !status.Signaled() {
		return "", false
	}
	return status.Signal().String(), true
}

// reportsOwnProcessGroup tells whether configureProcessGroup already isolated
// the command, so cleanup only signals groups this process created.
func reportsOwnProcessGroup(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid
}
