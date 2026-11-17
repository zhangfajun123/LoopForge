package runner

import (
	"context"
	"os/exec"
	"strings"
	stdsync "sync"
	"time"
)

// agentCLIFlagCache memoises, per binary path, whether a CLI accepts a flag. The
// probe runs once per machine per binary because it costs one process spawn and
// the answer only changes when the CLI is upgraded.
var agentCLIFlagCache stdsync.Map // key: binary + "\x00" + flag → bool

// agentCLIAcceptsFlag reports whether `binary <subcommand> --help` lists flag.
// A probe failure answers false, so callers use the argv shape that at least
// runs: a wrong "supported" answer kills the task, a wrong "unsupported" only
// loses the reasoning-effort setting.
func (d *Daemon) agentCLIAcceptsFlag(binary, subcommand, flag string) bool {
	if strings.TrimSpace(binary) == "" {
		return false
	}
	key := binary + "\x00" + flag
	if cached, ok := agentCLIFlagCache.Load(key); ok {
		return cached.(bool)
	}
	supported := helpListsFlag(binary, subcommand, flag)
	agentCLIFlagCache.Store(key, supported)
	return supported
}

func helpListsFlag(binary, subcommand, flag string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := make([]string, 0, 2)
	if strings.TrimSpace(subcommand) != "" {
		args = append(args, subcommand)
	}
	args = append(args, "--help")
	out, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if len(out) == 0 {
		return false
	}
	if err != nil && ctx.Err() != nil {
		// The probe timed out mid-write; the partial text cannot be trusted.
		return false
	}
	return mentionsFlag(string(out), flag)
}

// mentionsFlag matches a flag token, not a substring: opencode 2.x documents
// "provider/model#variant" in its help, which must not count as having
// --variant.
func mentionsFlag(help, flag string) bool {
	for _, line := range strings.Split(help, "\n") {
		idx := strings.Index(line, flag)
		if idx < 0 {
			continue
		}
		rest := line[idx+len(flag):]
		if rest == "" {
			return true
		}
		switch rest[0] {
		case ' ', '\t', ',', '=', '\r':
			return true
		}
	}
	return false
}