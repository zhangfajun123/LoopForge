package runtime

import "strings"

// WithOpenCodeStandalone inserts --standalone into `opencode run`'s flag block
// so the task starts a private server instead of using the user's shared managed
// service. Each task runs in its own private HOME, so two agents — or a service
// left behind by an earlier attempt — collide on opencode's fixed service port
// ("Managed service port 49374 ... is already in use"). Isolation is already the
// contract; --standalone is what makes it stick.
func WithOpenCodeStandalone(argv []string) []string {
	for _, arg := range argv {
		if arg == "--standalone" {
			return argv
		}
	}
	out := make([]string, 0, len(argv)+1)
	inserted := false
	for i, arg := range argv {
		out = append(out, arg)
		if !inserted && i > 0 && arg == "run" {
			out = append(out, "--standalone")
			inserted = true
		}
	}
	if !inserted {
		return argv
	}
	return out
}
// FoldOpenCodeVariant adapts an "--variant <effort>" pair to the local opencode
// CLI. opencode 2.x dropped the standalone flag and reads the reasoning effort
// from the model id instead (provider/model#variant), so an argv built for 1.x
// dies with "Unrecognized flag: --variant" before the turn even starts.
//
// standaloneSupported is the answer to "does this binary's `run` command list
// --variant?" — probed on the machine that is about to execute, because the
// version differs per runner. When the flag is unsupported the variant folds
// into the model argument; with no model argument there is nowhere to put it,
// so the pair is dropped rather than failing the task.
func FoldOpenCodeVariant(argv []string, standaloneSupported bool) []string {
	if standaloneSupported {
		return argv
	}
	variant, hasVariant := "", false
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		if argv[i] != "--variant" && !strings.HasPrefix(argv[i], "--variant=") {
			out = append(out, argv[i])
			continue
		}
		hasVariant = true
		if strings.HasPrefix(argv[i], "--variant=") {
			variant = strings.TrimPrefix(argv[i], "--variant=")
			continue
		}
		if i+1 < len(argv) {
			variant = argv[i+1]
			i++
		}
	}
	if !hasVariant || strings.TrimSpace(variant) == "" {
		return out
	}
	for i := 0; i < len(out); i++ {
		if out[i] != "-m" && out[i] != "--model" {
			continue
		}
		if i+1 >= len(out) || strings.TrimSpace(out[i+1]) == "" || strings.Contains(out[i+1], "#") {
			return out
		}
		out[i+1] = out[i+1] + "#" + variant
		return out
	}
	return out
}