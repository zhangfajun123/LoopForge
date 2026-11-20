package runner

import (
	"context"
	"strings"
	stdsync "sync"
	"time"

	flowruntime "agentflow/internal/runtime"
)

// Agent CLIs drift: a flag a role depends on can disappear between the version a
// machine carries and the version the platform's argv was written for. The runner
// is the only place that knows which binary is about to run, so it answers three
// questions once per binary and caches them: what help does it print, does it
// list a flag, and what version is it.
var (
	agentCLIHelpCache    stdsync.Map // key: binary + "\x00" + subcommand → string
	agentCLIFlagCache    stdsync.Map // key: binary + "\x00" + flag → bool
	agentCLIVersionCache stdsync.Map // key: binary → string
)

// agentCLIHelpText returns the help of `binary <subcommand>` plus the top-level
// help, which is where some CLIs document shared flags. An empty answer means
// nothing could be probed, and callers must not treat that as "unsupported".
func (d *Daemon) agentCLIHelpText(binary, subcommand string) string {
	if strings.TrimSpace(binary) == "" {
		return ""
	}
	key := binary + "\x00" + subcommand
	if cached, ok := agentCLIHelpCache.Load(key); ok {
		return cached.(string)
	}
	parts := make([]string, 0, 2)
	if strings.TrimSpace(subcommand) != "" {
		if out := runCLIProbe(binary, subcommand, "--help"); out != "" {
			parts = append(parts, out)
		}
	}
	if out := runCLIProbe(binary, "--help"); out != "" {
		joined := strings.Join(parts, "\n")
		if !strings.Contains(joined, out) {
			parts = append(parts, out)
		}
	}
	text := strings.Join(parts, "\n")
	agentCLIHelpCache.Store(key, text)
	return text
}

// agentCLIAcceptsFlag reports whether the CLI lists flag in its help. A failed
// probe answers false, so callers fall back to the argv shape that at least
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
	supported := mentionsFlag(d.agentCLIHelpText(binary, subcommand), flag)
	agentCLIFlagCache.Store(key, supported)
	return supported
}

// agentCLIVersionText is the version a failure message can quote, so an operator
// knows which machine's CLI is out of step.
func (d *Daemon) agentCLIVersionText(binary string) string {
	if strings.TrimSpace(binary) == "" {
		return ""
	}
	if cached, ok := agentCLIVersionCache.Load(binary); ok {
		return cached.(string)
	}
	version := ""
	if out := runCLIProbe(binary, "--version"); out != "" {
		version = strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	}
	agentCLIVersionCache.Store(binary, version)
	return version
}

// helpListsFlag keeps the narrow "does this binary list that flag" question
// answerable without a Daemon, which is how the probes are tested.
func helpListsFlag(binary, subcommand, flag string) bool {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(subcommand) != "" {
		if out := runCLIProbe(binary, subcommand, "--help"); out != "" {
			parts = append(parts, out)
		}
	}
	if out := runCLIProbe(binary, "--help"); out != "" {
		parts = append(parts, out)
	}
	return mentionsFlag(strings.Join(parts, "\n"), flag)
}

func runCLIProbe(binary string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	argv := append([]string{binary}, args...)
	out, err := flowruntime.NewCommandContext(ctx, argv...).CombinedOutput()
	if len(out) == 0 {
		return ""
	}
	if err != nil && ctx.Err() != nil {
		// The probe timed out mid-write; partial text cannot be trusted.
		return ""
	}
	return string(out)
}

// missingCLIFlags lists the long flags argv passes that help does not document.
// Positional arguments, flag values and short flags are not judged: help is the
// documented contract and only long flags are reliably listed there. Empty help
// means nothing can be judged, so nothing is reported.
func missingCLIFlags(argv []string, help string) []string {
	if strings.TrimSpace(help) == "" || len(argv) == 0 {
		return nil
	}
	seen := map[string]bool{}
	missing := make([]string, 0, 2)
	for _, arg := range argv[1:] {
		if !strings.HasPrefix(arg, "--") || len(arg) <= 2 {
			// Binary, subcommand, short flags and positional prompt text.
			continue
		}
		flag := arg
		if idx := strings.Index(flag, "="); idx > 0 {
			flag = flag[:idx]
		}
		if seen[flag] {
			continue
		}
		seen[flag] = true
		if !mentionsFlag(help, flag) {
			missing = append(missing, flag)
		}
	}
	return missing
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
