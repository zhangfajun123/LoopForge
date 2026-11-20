//go:build windows

package runtime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Windows counterpart of workspace_terminal_sandbox.go (!windows). The
// environment-identity helpers are referenced by the untagged
// workspace_preparation.go, so the windows/amd64 runner and gate-mcp builds
// need them even though the sandbox terminal pool and the unix filesystem
// identity (syscall.Stat_t) do not exist on this platform.

// SandboxEnvironmentIdentity identifies an execution environment generation.
// Workspace preparation stamps must include this value: a bind-mounted file
// stamp survives container recreation, while installed tools and processes do
// not. Values are hashed so configured environment secrets never enter receipts.
func SandboxEnvironmentIdentity(ctx context.Context, sandbox Sandbox) (string, error) {
	switch s := sandbox.(type) {
	case *managedSandbox:
		return SandboxEnvironmentIdentity(ctx, s.Sandbox)
	case *dockerSandbox:
		if err := s.ensureStarted(ctx); err != nil {
			return "", err
		}
		identity, err := ReadSandboxEnvironmentIdentity(ctx, s)
		if err == nil || ctx.Err() != nil {
			return identity, err
		}
		// The started flag is a process-local cache. A container stopped or
		// removed outside this process (daemon restart, prune, manual cleanup)
		// must be revalidated here, or every later preparation keeps failing.
		// Restarting or recreating changes the identity, so preparation replays.
		s.invalidate()
		if err := s.ensureStarted(ctx); err != nil {
			return "", err
		}
	}
	return ReadSandboxEnvironmentIdentity(ctx, sandbox)
}

// ReadSandboxEnvironmentIdentity inspects an existing execution environment
// without creating or starting a container. Read-only status/peek paths use it
// to validate a warm preparation receipt while another group member is running.
func ReadSandboxEnvironmentIdentity(ctx context.Context, sandbox Sandbox) (string, error) {
	switch s := sandbox.(type) {
	case *managedSandbox:
		return ReadSandboxEnvironmentIdentity(ctx, s.Sandbox)
	case *dockerSandbox:
		output, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}|{{.Id}}|{{.State.StartedAt}}", s.name).Output()
		if err != nil {
			return "", fmt.Errorf("inspect workspace environment: %w", err)
		}
		if !strings.HasPrefix(strings.TrimSpace(string(output)), "true|") {
			return "", fmt.Errorf("workspace execution environment is not running")
		}
		overlay, err := json.Marshal(s.profile.Environment)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(append(output, overlay...))
		return "docker:" + hex.EncodeToString(sum[:]), nil
	case *localSandbox:
		return filesystemEnvironmentIdentity(s.Workspace(), s.Driver(), s.environment)
	case *processSandbox:
		return filesystemEnvironmentIdentity(s.Workspace(), s.Driver(), s.profile.Environment)
	default:
		return "", fmt.Errorf("unsupported workspace environment identity")
	}
}

// filesystemEnvironmentIdentity: windows has no syscall.Stat_t, so the stable
// workspace path stands in for the unix device/inode pair. The identity still
// changes whenever the driver, the configured environment or the process epoch
// changes, which is what preparation reuse keys on.
func filesystemEnvironmentIdentity(workspace, driver string, environment map[string]string) (string, error) {
	if _, err := os.Stat(workspace); err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Driver       string
		Workspace    string
		Environment  map[string]string
		ProcessEpoch string
	}{driver, filepath.Clean(workspace), environment, terminalProcessEnvironmentEpoch()})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return driver + ":" + hex.EncodeToString(sum[:]), nil
}

// Local tool installs may persist on disk, but background services die when
// this execution process restarts. Share one epoch across handles in a process,
// and force preparation to recheck after server/runner restart.
var terminalProcessEnvironmentEpoch = sync.OnceValue(func() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
})
