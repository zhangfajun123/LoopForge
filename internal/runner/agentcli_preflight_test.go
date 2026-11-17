package runner

import "testing"

// Preflight judges the argv the platform built against the help the executing
// machine prints: every long flag the CLI no longer documents is reported before
// a turn spends tokens.
func TestMissingCLIFlags(t *testing.T) {
	opencodeV1 := `FLAGS
  --format choice         Output format (choices: default, json)
  --thinking              Show thinking blocks
  --variant               model variant (provider-specific reasoning effort)
  -m, --model string      Model to use`
	opencodeV2 := `FLAGS
  --format choice         Output format (choices: default, json)
  --thinking              Show thinking blocks
  --standalone            Run with a private server
  -m, --model string      Model to use in the format provider/model#variant`

	cases := []struct {
		name string
		argv []string
		help string
		want []string
	}{
		{
			name: "flags the CLI documents pass",
			argv: []string{"opencode", "run", "--format", "json", "--thinking", "prompt"},
			help: opencodeV1,
			want: nil,
		},
		{
			name: "a flag this version dropped is reported",
			argv: []string{"opencode", "run", "--format", "json", "--variant", "high", "prompt"},
			help: opencodeV2,
			want: []string{"--variant"},
		},
		{
			name: "inline values are judged by flag name",
			argv: []string{"opencode", "run", "--variant=high", "prompt"},
			help: opencodeV2,
			want: []string{"--variant"},
		},
		{
			name: "short flags and the prompt are not judged",
			argv: []string{"opencode", "run", "-m", "agentflow/m", "-s", "ses_1", "--format", "json"},
			help: opencodeV2,
			want: nil,
		},
		{
			name: "every undocumented flag is listed once",
			argv: []string{"opencode", "run", "--variant", "high", "--obsolete", "--variant", "low"},
			help: opencodeV2,
			want: []string{"--variant", "--obsolete"},
		},
		{
			name: "no help means nothing is judged",
			argv: []string{"opencode", "run", "--variant", "high"},
			help: "",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := missingCLIFlags(tc.argv, tc.help)
			if len(got) != len(tc.want) {
				t.Fatalf("missing = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("missing = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// The preflight must not fire on a machine whose CLI it cannot probe: an empty
// probe answer is "unknown", not "unsupported", or every docker sandbox task
// would be refused.
func TestMissingCLIFlagsSkipsUnprobableBinaries(t *testing.T) {
	if got := missingCLIFlags([]string{"/nonexistent/opencode", "run", "--variant", "high"}, ""); got != nil {
		t.Fatalf("unprobed binary reported %v", got)
	}
}