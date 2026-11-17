package runner

import (
	"strings"
	"testing"

	"agentflow/internal/domain"
	"agentflow/internal/runtime"
)

// The remote runner binds Codex credentials itself. Without an explicit model
// Codex falls back to its built-in default, which the self-hosted gateway
// answers with 404 Model not found, so the platform sends the resolved model.
func TestBindInvocationCredentialsAppliesCodexModel(t *testing.T) {
	inv := &domain.RunnerInvocation{
		CredentialSource: domain.RuntimeAuthPersonal,
		CodexAPIKey:      "sk-test",
		CodexBaseURL:     "https://gw.example/v1",
		CodexModel:       "deepseek-v4.1-flash-1017",
	}
	argv, env, err := bindInvocationCredentials(inv, []runtime.CredentialBinding{runtime.BindCodex}, []string{"codex", "exec", "do it"})
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if env["CODEX_API_KEY"] != "sk-test" {
		t.Fatalf("CODEX_API_KEY not injected: %q", env["CODEX_API_KEY"])
	}
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, `model="deepseek-v4.1-flash-1017"`) {
		t.Fatalf("resolved model missing from argv: %s", joined)
	}
	if argv[len(argv)-1] != "do it" {
		t.Fatalf("prompt must stay last: %#v", argv)
	}
}

func TestBindInvocationCredentialsKeepsExplicitCodexModel(t *testing.T) {
	inv := &domain.RunnerInvocation{
		CredentialSource: domain.RuntimeAuthPersonal,
		CodexAPIKey:      "sk-test",
		CodexBaseURL:     "https://gw.example/v1",
		CodexModel:       "deepseek-v4.1-flash-1017",
	}
	argv, _, err := bindInvocationCredentials(inv, []runtime.CredentialBinding{runtime.BindCodex}, []string{"codex", "exec", "--model", "glm-5.3-flash", "do it"})
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if strings.Contains(strings.Join(argv, " "), "deepseek-v4.1-flash-1017") {
		t.Fatalf("resolved model overrode the explicit one: %#v", argv)
	}
}

// Local credential mode leaves the machine's own login alone: no personal key
// and no provider override, so the resolved model must not be injected either.
func TestBindInvocationCredentialsSkipsCodexBindingInLocalMode(t *testing.T) {
	inv := &domain.RunnerInvocation{CredentialSource: domain.RuntimeAuthLocal, CodexModel: "deepseek-v4.1-flash-1017"}
	argv, env, err := bindInvocationCredentials(inv, []runtime.CredentialBinding{runtime.BindCodex}, []string{"codex", "exec", "do it"})
	if err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if env["CODEX_API_KEY"] != "" {
		t.Fatalf("local mode injected a key: %q", env["CODEX_API_KEY"])
	}
	if len(argv) != 3 {
		t.Fatalf("local mode rewrote argv: %#v", argv)
	}
}