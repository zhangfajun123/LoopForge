//go:build windows

package runtime

import (
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

// createNewProcessGroup is CREATE_NEW_PROCESS_GROUP. Windows has no POSIX
// process groups; the flag only makes CTRL_BREAK reachable and lets a watcher
// identify the subtree that shares the leader.
const createNewProcessGroup = 0x00000200

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNewProcessGroup
}

func configureCommandCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		terminateProcess(cmd.Process.Pid)
		return nil
	}
}

// killProcessGroup terminates the process tree below a started command.
// Windows cannot signal a POSIX group, so a started command is ended with
// taskkill /T to also take its children (Agent CLI wrappers, MCP servers).
// A plain TerminateProcess remains the fallback when taskkill is unavailable.
func killProcessGroup(cmd *exec.Cmd, force bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if !killTreeWithTaskkill(cmd.Process.Pid) {
		terminateProcess(cmd.Process.Pid)
	}
}

// killProcessTree force-terminates pid and its children.
func killProcessTree(pid int) {
	if pid <= 0 {
		return
	}
	if killTreeWithTaskkill(pid) {
		return
	}
	terminateProcess(pid)
}

func killTreeWithTaskkill(pid int) bool {
	taskkill, err := exec.LookPath("taskkill")
	if err != nil {
		return false
	}
	return exec.Command(taskkill, "/T", "/F", "/PID", strconv.Itoa(pid)).Run() == nil
}

func terminateProcess(pid int) {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_ = windows.TerminateProcess(handle, 1)
}

// processAlive reports whether pid names a live process. Windows exposes no
// signal-0 equivalent; opening the process with a minimal access right is the
// canonical liveness probe.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return true
	}
	const stillActive = 259
	return code == stillActive
}

// exitStatusSignal reports the terminating signal name for a process state
// that was signalled, and whether the platform records one. Windows process
// states only carry an exit code.
func exitStatusSignal(interface{ Sys() any }) (string, bool) {
	return "", false
}

// reportsOwnProcessGroup tells whether configureProcessGroup already isolated
// the command, so cleanup only signals groups this process created.
func reportsOwnProcessGroup(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.CreationFlags&createNewProcessGroup != 0
}
