package runtime

import (
	"reflect"
	"testing"
)

func TestWithOpenCodeStandalone(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "inserted right after the subcommand",
			argv: []string{"opencode", "run", "--format", "json", "-m", "agentflow/m", "prompt"},
			want: []string{"opencode", "run", "--standalone", "--format", "json", "-m", "agentflow/m", "prompt"},
		},
		{
			name: "already standalone stays put",
			argv: []string{"opencode", "run", "--standalone", "--format", "json", "prompt"},
			want: []string{"opencode", "run", "--standalone", "--format", "json", "prompt"},
		},
		{
			name: "no subcommand means no change",
			argv: []string{"opencode", "--format", "json", "prompt"},
			want: []string{"opencode", "--format", "json", "prompt"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WithOpenCodeStandalone(tc.argv); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("argv = %v, want %v", got, tc.want)
			}
		})
	}
}

// opencode 2.x has no standalone --variant flag: the reasoning effort has to
// ride on the model id. Both shapes must survive the fold unchanged.
func TestFoldOpenCodeVariant(t *testing.T) {
	cases := []struct {
		name      string
		argv      []string
		supported bool
		want      []string
	}{
		{
			name:      "1.x keeps the standalone flag",
			argv:      []string{"opencode", "run", "-m", "agentflow/m", "--variant", "high", "do it"},
			supported: true,
			want:      []string{"opencode", "run", "-m", "agentflow/m", "--variant", "high", "do it"},
		},
		{
			name:      "2.x folds the effort into the model",
			argv:      []string{"opencode", "run", "-m", "agentflow/m", "--variant", "high", "do it"},
			supported: false,
			want:      []string{"opencode", "run", "-m", "agentflow/m#high", "do it"},
		},
		{
			name:      "2.x fold with the long model flag",
			argv:      []string{"opencode", "run", "--model", "agentflow/m", "--variant=high", "do it"},
			supported: false,
			want:      []string{"opencode", "run", "--model", "agentflow/m#high", "do it"},
		},
		{
			name:      "2.x without a model has nowhere to put it",
			argv:      []string{"opencode", "run", "--variant", "high", "do it"},
			supported: false,
			want:      []string{"opencode", "run", "do it"},
		},
		{
			name:      "2.x leaves a model that already carries a variant",
			argv:      []string{"opencode", "run", "-m", "agentflow/m#max", "--variant", "high", "do it"},
			supported: false,
			want:      []string{"opencode", "run", "-m", "agentflow/m#max", "do it"},
		},
		{
			name:      "no variant means no work",
			argv:      []string{"opencode", "run", "-m", "agentflow/m", "do it"},
			supported: false,
			want:      []string{"opencode", "run", "-m", "agentflow/m", "do it"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FoldOpenCodeVariant(tc.argv, tc.supported); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("argv = %v, want %v", got, tc.want)
			}
		})
	}
}