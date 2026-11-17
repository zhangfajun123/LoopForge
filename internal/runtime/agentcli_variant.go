package runtime

import "strings"

// FoldOpenCodeVariant adapts an "--variant <effort>" pair to the local opencode
// CLI. opencode 2.x removed the standalone flag and reads the reasoning effort
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