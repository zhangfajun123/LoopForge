package runtime

import (
	"slices"
	"strings"
	"testing"

	"agentflow/internal/domain"
)

// A role that never picked a model leaves Codex on its built-in default, which a
// self-hosted gateway does not host: the run fails with 404 Model not found. The
// binding therefore falls back to the model the user saved for Codex.
func TestBindCodexFillsTheModelWhenTheRoleHasNone(t *testing.T) {
	spec := CommandSpec{
		Argv:           []string{"codex", "exec", "--json", "--skip-git-repo-check", "do it"},
		Provider:       "codex",
		CredentialBindings: []CredentialBinding{BindCodex},
	}
	credentials := RunCredentials{Codex: ProviderCredential{
		APIKey:  "sk-test",
		BaseURL: "https://gw.example/v1",
		Models:  []domain.CLIModel{{ID: "glm-5.3-flash"}, {ID: "deepseek-v4.1-flash-1017"}},
	}}

	workspace := t.TempDir()
	bound, err := bindCommand(&localSandbox{workspace: workspace}, workspace, spec, credentials)
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	joined := strings.Join(bound.Argv, " ")
	if !strings.Contains(joined, "model=") || !strings.Contains(joined, "glm-5.3-flash") {
		t.Fatalf("fallback model missing from argv: %s", joined)
	}
	if bound.Argv[len(bound.Argv)-1] != "do it" {
		t.Fatalf("prompt must stay last: %#v", bound.Argv)
	}
}

func TestBindCodexKeepsAnExplicitModel(t *testing.T) {
	spec := CommandSpec{
		Argv:               []string{"codex", "exec", "--json", "-m", "gpt-5.5", "do it"},
		Provider:           "codex",
		CredentialBindings: []CredentialBinding{BindCodex},
	}
	credentials := RunCredentials{Codex: ProviderCredential{
		APIKey:  "sk-test",
		BaseURL: "https://gw.example/v1",
		Models:  []domain.CLIModel{{ID: "glm-5.3-flash"}},
	}}
	workspace := t.TempDir()
	bound, err := bindCommand(&localSandbox{workspace: workspace}, workspace, spec, credentials)
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if !slices.Contains(bound.Argv, "gpt-5.5") {
		t.Fatalf("explicit model was dropped: %#v", bound.Argv)
	}
	if strings.Contains(strings.Join(bound.Argv, " "), "glm-5.3-flash") {
		t.Fatalf("fallback model overrode the explicit one: %#v", bound.Argv)
	}
}

// The same gap exists on the agent-CLI path (hermes/opencode/dsh): a role with
// no model used to be rejected, now it resolves to the saved catalog.
func TestApplyAgentCLICredentialResolvesDefaultModel(t *testing.T) {
	home := HostAgentCLIHome(t.TempDir())
	cred := ProviderCredential{
		APIKey:  "sk-test",
		BaseURL: "https://gw.example/v1",
		Models:  []domain.CLIModel{{ID: "deepseek-v4.1-flash-1017"}},
	}
	if _, _, err := ApplyAgentCLICredential(AgentCLIHermes, cred, AgentCLIOptions{Model: "default"}, home, "/w/home", nil); err != nil {
		t.Fatalf("hermes rejected the default model: %v", err)
	}
}