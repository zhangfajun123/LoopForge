//go:build !windows

package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	flowruntime "agentflow/internal/runtime"
)

// setProcessGroup puts the CLI and all of its children into a new process
// group so a cancel cannot leave orphaned shells or MCP servers behind.
func setProcessGroup(cmd *exec.Cmd) {
	flowruntime.ConfigureProcessGroup(cmd)
}

// terminateProcessGroup sends SIGTERM to the whole group, waits a grace
// period, then escalates to SIGKILL.
// processDone belongs to the single Wait caller; termination must not reap it.
func terminateProcessGroup(cmd *exec.Cmd, processDone <-chan struct{}) {
	select {
	case <-processDone:
		return
	default:
	}
	if cmd.Process == nil {
		return
	}
	flowruntime.SignalProcessGroup(cmd, false)
	timer := time.NewTimer(cancelGrace)
	defer timer.Stop()
	select {
	case <-processDone:
	case <-timer.C:
		flowruntime.SignalProcessGroup(cmd, true)
	}
}

// platformCommandContext builds the direct child process. POSIX does not need
// a shell wrapper: a resolvable interpreter path is executed as-is.
func platformCommandContext(ctx context.Context, argv []string) *exec.Cmd {
	return exec.CommandContext(ctx, argv[0], argv[1:]...)
}

// lookupExecutableInDir resolves name inside dir. Unix binaries carry an
// executable bit, so the permission mask is the only test.
func lookupExecutableInDir(dir, name string) (string, bool) {
	candidate := filepath.Join(dir, name)
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", false
	}
	return candidate, true
}
