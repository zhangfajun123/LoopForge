//go:build windows

package runtime

import (
	"context"
	"errors"
	"os"

	"agentflow/internal/domain"
)

type TerminalPool struct{}

type TerminalResult struct {
	ID     string `json:"id"`
	Data   []byte `json:"data,omitempty"`
	Exited bool   `json:"exited"`
}

type TerminalGitCredentials struct {
	HTTPS   map[string]domain.GitHTTPExecutionCredential
	SSHKeys map[string]string
}

func terminalBaseEnvironment(home string) []string {
	return []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "TERM=xterm-256color", "LANG=C.UTF-8"}
}

func (*TerminalPool) Handle(string, string, domain.WorkspaceTerminalRequest, func()) (TerminalResult, error) {
	return TerminalResult{}, errors.New("interactive workspace terminals are not supported on Windows runners")
}

func (*TerminalPool) HandleSandbox(context.Context, Sandbox, string, domain.WorkspaceTerminalRequest, func()) (TerminalResult, error) {
	return TerminalResult{}, errors.New("interactive workspace terminals are not supported on Windows runners")
}

func (*TerminalPool) HandleSandboxWithGit(context.Context, Sandbox, string, domain.WorkspaceTerminalRequest, func(), TerminalGitCredentials) (TerminalResult, error) {
	return TerminalResult{}, errors.New("interactive workspace terminals are not supported on Windows runners")
}

func (*TerminalPool) Active() bool                 { return false }
func (*TerminalPool) HasRunScope(string) bool      { return false }
func (*TerminalPool) HasScope(string) bool         { return false }
func (*TerminalPool) CloseScope(string)            {}
