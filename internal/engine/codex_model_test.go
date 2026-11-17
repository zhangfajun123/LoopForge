package engine

import (
	"testing"

	"agentflow/internal/domain"
)

// A role that never picked a model leaves Codex on its built-in default, which a
// self-hosted gateway answers with 404 Model not found.
func TestResolvedCodexModel(t *testing.T) {
	saved := []domain.CLIModel{{ID: "default"}, {ID: "deepseek-v4.1-flash-1017"}}
	cases := []struct {
		name      string
		roleModel string
		models    []domain.CLIModel
		want      string
	}{
		{"role model wins", "glm-5.3-flash", saved, "glm-5.3-flash"},
		{"empty role model uses the saved catalog", "", saved, "deepseek-v4.1-flash-1017"},
		{"literal default uses the saved catalog", "default", saved, "deepseek-v4.1-flash-1017"},
		{"no saved models stays empty", "", nil, ""},
	}
	for _, c := range cases {
		if got := resolvedCodexModel(c.roleModel, c.models); got != c.want {
			t.Errorf("%s: resolvedCodexModel(%q) = %q, want %q", c.name, c.roleModel, got, c.want)
		}
	}
}