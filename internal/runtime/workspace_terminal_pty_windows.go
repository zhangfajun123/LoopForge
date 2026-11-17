//go:build windows

package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// terminalSession is the duplex byte stream between the pool and the shell.
// The PTY driver (creack/pty) does not support Windows, so the Windows
// implementation of a terminal is a pipe-backed command session: stdin, and
// a single merged stdout/stderr stream. It supports typing commands, reading
// output and closing; it is not a full ConPTY (no window size adaptation, no
// alt-screen programs).
type terminalSession interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

type pipeTerminalSession struct {
	once sync.Once
	in   *os.File
	out  *os.File
}

func (s *pipeTerminalSession) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *pipeTerminalSession) Write(p []byte) (int, error) { return s.in.Write(p) }
func (s *pipeTerminalSession) Close() error {
	var err error
	s.once.Do(func() {
		if s.in != nil {
			err = s.in.Close()
		}
		if s.out != nil {
			if closeErr := s.out.Close(); err == nil {
				err = closeErr
			}
		}
	})
	return err
}

// terminalShellArgv starts an interactive PowerShell that reads commands from
// its stdin pipe. -Command - is the documented stdin-hosted command stream.
func terminalShellArgv(string) []string {
	return []string{"powershell.exe", "-NoLogo", "-NoProfile", "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", "-"}
}

func startTerminalSession(cmd *exec.Cmd, cols, rows uint16) (terminalSession, error) {
	stdin, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdout, stdoutWriter, err := os.Pipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdinWriter.Close()
		return nil, err
	}
	cmd.Stdin = stdin
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stdoutWriter
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdinWriter.Close()
		_ = stdout.Close()
		_ = stdoutWriter.Close()
		return nil, err
	}
	// The child owns its ends now; leaving them open in the parent would keep
	// the pipes from reporting EOF when the shell exits.
	_ = stdin.Close()
	_ = stdoutWriter.Close()
	return &pipeTerminalSession{in: stdinWriter, out: stdout}, nil
}

// resizeTerminalSession is a no-op: a pipe session has no window size.
func resizeTerminalSession(_ terminalSession, cols, rows uint16) error {
	return nil
}

func stopTerminalSession(session terminalSession, cmd *exec.Cmd) {
	if session != nil {
		_ = session.Close()
	}
	if cmd == nil || cmd.Process == nil {
		return
	}
	if !killTreeWithTaskkill(cmd.Process.Pid) {
		if err := cmd.Process.Kill(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "close terminal process:", err)
		}
	}
}

// terminalPlatformEnvironment keeps the Windows variables PowerShell needs to
// load modules and resolve built-ins. The private home replaces the operator's
// profile, so USERPROFILE/TEMP/TMP point into it.
func terminalPlatformEnvironment(home string) []string {
	env := []string{"USERPROFILE=" + home, "TEMP=" + home, "TMP=" + home}
	for _, key := range []string{"SystemRoot", "WINDIR", "COMSPEC", "PATHEXT", "PSModulePath", "APPDATA", "LOCALAPPDATA"} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}
