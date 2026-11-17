package httpapi

import (
	"strings"
	"testing"

	"agentflow/internal/domain"
	"agentflow/internal/runtime"
)

// Chat's 「CLI default」 is stored as the literal "default"; passing it through
// made the CLI ask the gateway for a model named default (404 Model not found),
// which is why a host-side opencode shim used to rewrite the name.
func TestAssistantAgentCLIModelResolvesCLIDefault(t *testing.T) {
	saved := runtime.ProviderCredential{Models: []domain.CLIModel{{ID: "deepseek-v4.1-flash-1017"}, {ID: "glm-5.3-flash"}}}

	cases := []struct {
		name  string
		kind  string
		model string
		cred  runtime.ProviderCredential
		want  string
	}{
		{"default uses the saved catalog", "opencode", "default", saved, "deepseek-v4.1-flash-1017"},
		{"empty uses the saved catalog", "opencode", "  ", saved, "deepseek-v4.1-flash-1017"},
		{"DEFAULT is case insensitive", "hermes", "Default", runtime.ProviderCredential{Models: []domain.CLIModel{{ID: "glm-5.3-flash"}}}, "glm-5.3-flash"},
		{"a real model is kept", "opencode", "glm-5.3", saved, "glm-5.3"},
		{"falls back to the builtin catalog default", "codebuddy", "default", runtime.ProviderCredential{}, assistantKnownCatalog().Default},
		{"no catalog means the CLI default", "opencode", "default", runtime.ProviderCredential{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := assistantAgentCLIModel(c.kind, c.model, c.cred); got != c.want {
				t.Fatalf("assistantAgentCLIModel(%q, %q) = %q, want %q", c.kind, c.model, got, c.want)
			}
		})
	}
}

// The literal "default" must never reach the CLI: opencode would be told
// "-m agentflow/default" and the gateway answers 404.
func TestAgentCLICommandNeverPassesTheDefaultLiteral(t *testing.T) {
	spec := runtime.AgentCLICommand(runtime.AgentCLIOpenCode, runtime.AgentCLICommandOptions{Prompt: "hi", Model: "default"})
	for _, arg := range spec.Argv {
		if strings.Contains(arg, "agentflow/default") || arg == "default" {
			t.Fatalf("argv leaks the literal default: %#v", spec.Argv)
		}
	}
}