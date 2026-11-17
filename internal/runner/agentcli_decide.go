package runner

import (
	"os"

	"agentflow/internal/runtime"
)

// agentCLIDecision is what this machine concluded about an agent CLI invocation.
// It exists so the same facts can be unit-tested and traced, instead of being
// buried in the task-execution flow.
type agentCLIDecision struct {
	Provider            string
	Driver              string
	Sandboxed           bool
	LocalBinary         bool
	Probed              bool
	VariantSupported    bool
	StandaloneSupported bool
}

// decideAgentCLIArgv applies this machine's opencode adaptations to argv and
// reports what it decided.
//
// opencode versions disagree about two things the platform's argv depends on:
// 2.x dropped the standalone --variant flag (the effort rides on the model id),
// and 2.x's `run` is a client for a managed service on a fixed port, which every
// task in its own private HOME fights over. Both adaptations are applied only
// when argv[0] exists on this host: otherwise the CLI belongs to a sandbox image
// and probing it would describe a different binary.
func (d *Daemon) decideAgentCLIArgv(provider string, sandboxed bool, driver string, argv []string) ([]string, agentCLIDecision) {
	decision := agentCLIDecision{Provider: provider, Driver: driver, Sandboxed: sandboxed}
	if provider != runtime.AgentCLIOpenCode || len(argv) == 0 {
		return argv, decision
	}
	if info, err := os.Stat(argv[0]); err == nil && !info.IsDir() {
		decision.LocalBinary = true
	}
	help := d.agentCLIHelpText(argv[0], "run")
	decision.Probed = len(help) > 0
	decision.VariantSupported = mentionsFlag(help, "--variant")
	decision.StandaloneSupported = mentionsFlag(help, "--standalone")
	if !decision.LocalBinary {
		return argv, decision
	}
	argv = runtime.FoldOpenCodeVariant(argv, decision.VariantSupported)
	if decision.StandaloneSupported {
		argv = runtime.WithOpenCodeStandalone(argv)
	}
	return argv, decision
}