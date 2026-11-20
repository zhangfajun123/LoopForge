//go:build windows

package runtime

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// ConfigureProcessTree arranges for cancellation to stop the process and its
// descendants. Windows has no Unix-style process groups, so taskkill owns the
// tree termination.
func ConfigureProcessTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x00000200 // CREATE_NEW_PROCESS_GROUP
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return SignalProcessTree(cmd.Process.Pid, true)
	}
}

func ProcessTreeConfigured(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.CreationFlags&0x00000200 != 0
}

// SignalProcessTree terminates a Windows process and its descendants.
func SignalProcessTree(pid int, force bool) error {
	args := []string{"/PID", strconv.Itoa(pid), "/T", "/F"}
	err := exec.Command("taskkill.exe", args...).Run()
	if err == nil {
		return nil
	}
	return fmt.Errorf("taskkill process tree %d: %w", pid, err)
}

// ProcessAlive checks the process table without requiring elevated privileges.
func ProcessAlive(pid int) bool {
	out, err := exec.Command("tasklist.exe", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), fmt.Sprintf("\"%d\"", pid))
}
