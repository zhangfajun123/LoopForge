//go:build !windows

package runtime

import (
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// terminalSession is the duplex byte stream between the pool and the shell.
// On Unix it is the PTY slave; on Windows a pair of pipes stands in for it.
type terminalSession interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

// terminalShellArgv runs a POSIX login-capable shell inside the sandbox.
func terminalShellArgv(script string) []string {
	return []string{"/bin/sh", "-c", script}
}

func startTerminalSession(cmd *exec.Cmd, cols, rows uint16) (terminalSession, error) {
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	return file, nil
}

func resizeTerminalSession(session terminalSession, cols, rows uint16) error {
	file, ok := session.(*os.File)
	if !ok {
		return nil
	}
	return pty.Setsize(file, &pty.Winsize{Cols: cols, Rows: rows})
}

// stopTerminalSession hangs up the controlling PTY and every process group the
// shell left behind, then closes the descriptor.
func stopTerminalSession(session terminalSession, cmd *exec.Cmd) {
	file, ok := session.(*os.File)
	if !ok {
		if session != nil {
			_ = session.Close()
		}
		return
	}
	fg, _ := unix.IoctlGetInt(int(file.Fd()), unix.TIOCGPGRP)
	if cmd != nil && cmd.Process != nil {
		// Closing the controlling PTY also hangs up the foreground group.
		_ = unix.Kill(-cmd.Process.Pid, unix.SIGHUP)
	}
	_ = file.Close()
	time.Sleep(100 * time.Millisecond)
	if cmd == nil || cmd.Process == nil {
		return
	}
	if fg > 0 {
		_ = unix.Kill(-fg, unix.SIGKILL)
	}
	_ = unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
}

// terminalPlatformEnvironment carries the variables a POSIX login shell needs
// that the minimal terminal environment does not inherit.
func terminalPlatformEnvironment(string) []string {
	return []string{"TERM=xterm-256color", "LANG=C.UTF-8"}
}
