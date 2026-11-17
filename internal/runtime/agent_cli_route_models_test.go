package runtime

import (
	"strings"
	"testing"

	"agentflow/internal/domain"
)

func routeCredential(baseURL string, models ...string) domain.AgentCLICredential {
	cred := domain.AgentCLICredential{APIKey: "test-key", BaseURL: baseURL, Protocol: domain.AgentCLIProtocolChat}
	for _, id := range models {
		cred.Models = append(cred.Models, domain.CLIModel{ID: id})
	}
	return cred
}

// Personal settings are edited per CLI, so a CLI added later on a route that
// already works arrives with an empty model list. The run used to fail
// 「Hermes 需要指定模型」 although the same gateway plainly serves models; the
// models belong to the route, not to one CLI.
func TestAgentCLIUserCredentialsInheritsModelsFromSameRoute(t *testing.T) {
	user := domain.User{AgentCLICredentials: map[string]domain.AgentCLICredential{
		AgentCLIOpenCode: routeCredential("https://gateway.example/v1", "deepseek-v4.1-flash-1017", "glm-5.3-flash"),
		AgentCLIHermes:   routeCredential("https://gateway.example/v1/"),
	}}
	creds := AgentCLIUserCredentials(user)
	got := creds[AgentCLIHermes].Models
	if len(got) != 2 || got[0].ID != "deepseek-v4.1-flash-1017" {
		t.Fatalf("hermes did not inherit the route's models: %+v", got)
	}
	if first := FirstCredentialModel(got); first != "deepseek-v4.1-flash-1017" {
		t.Fatalf("resolved model = %q", first)
	}
	if creds[AgentCLIHermes].APIKey != "test-key" || creds[AgentCLIHermes].BaseURL == "" {
		t.Fatal("inheriting models must not touch the CLI's own key or route")
	}
}

func TestAgentCLIUserCredentialsKeepsItsOwnModels(t *testing.T) {
	user := domain.User{AgentCLICredentials: map[string]domain.AgentCLICredential{
		AgentCLIOpenCode: routeCredential("https://gateway.example/v1", "glm-5.3-flash"),
		AgentCLIHermes:   routeCredential("https://gateway.example/v1", "deepseek-v4.1-flash-1017"),
	}}
	if got := AgentCLIUserCredentials(user)[AgentCLIHermes].Models; len(got) != 1 || got[0].ID != "deepseek-v4.1-flash-1017" {
		t.Fatalf("an explicit list was overwritten: %+v", got)
	}
}

// A different route serves different models; inheriting across routes would
// send a model the gateway there does not host.
func TestAgentCLIUserCredentialsIgnoresOtherRoutes(t *testing.T) {
	user := domain.User{AgentCLICredentials: map[string]domain.AgentCLICredential{
		AgentCLIOpenCode: routeCredential("https://gateway.example/v1", "glm-5.3-flash"),
		AgentCLIHermes:   routeCredential("https://other.example/v1"),
	}}
	if got := AgentCLIUserCredentials(user)[AgentCLIHermes].Models; len(got) != 0 {
		t.Fatalf("models leaked across routes: %+v", got)
	}
}

// The native route (no Base URL) has no sibling to inherit from.
func TestAgentCLIUserCredentialsNativeRouteInheritsNothing(t *testing.T) {
	user := domain.User{AgentCLICredentials: map[string]domain.AgentCLICredential{
		AgentCLIOpenCode: routeCredential("https://gateway.example/v1", "glm-5.3-flash"),
		AgentCLIHermes:   routeCredential(""),
	}}
	if got := AgentCLIUserCredentials(user)[AgentCLIHermes].Models; len(got) != 0 {
		t.Fatalf("the native route inherited a gateway model: %+v", got)
	}
}

// Two CLIs on the same route with different lists: the choice must not depend on
// map iteration order.
func TestAgentCLIUserCredentialsInheritsDeterministically(t *testing.T) {
	user := domain.User{AgentCLICredentials: map[string]domain.AgentCLICredential{
		AgentCLIOpenCode: routeCredential("https://gateway.example/v1", "opencode-model"),
		AgentCLIDeepSeek: routeCredential("https://gateway.example/v1", "dsh-model"),
		AgentCLIHermes:   routeCredential("https://gateway.example/v1"),
	}}
	for i := 0; i < 20; i++ {
		got := AgentCLIUserCredentials(user)[AgentCLIHermes].Models
		if len(got) != 1 || got[0].ID != "dsh-model" {
			t.Fatalf("inheritance is not deterministic (dsh sorts before opencode): %+v", got)
		}
	}
}

// With no model anywhere, the error has to say where to fix it.
func TestApplyAgentCLICredentialNamesTheModelGap(t *testing.T) {
	cred := ProviderCredential{APIKey: "test-key", BaseURL: "https://gateway.example/v1", Protocol: domain.AgentCLIProtocolChat}
	_, _, err := ApplyAgentCLICredential(AgentCLIHermes, cred, AgentCLIOptions{}, AgentCLIHome{}, "/tmp/home", []string{"hermes", "-z"})
	if err == nil {
		t.Fatal("a model-less Hermes invocation was accepted")
	}
	if !strings.Contains(err.Error(), "需要指定模型") || !strings.Contains(err.Error(), "个人配置") {
		t.Fatalf("unhelpful error: %v", err)
	}
}