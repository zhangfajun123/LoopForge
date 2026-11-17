package runtime

import (
	"strings"
	"testing"
)

func TestCodexUserProviderArgvDisablesHostedWebSearch(t *testing.T) {
	out, err := CodexUserProviderArgv([]string{"codex", "exec", "do it"}, "https://opengw.clouddeveloper.club/v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(out, " "), `web_search="disabled"`) {
		t.Fatalf("web_search not disabled: %v", out)
	}
	if out[len(out)-1] != "do it" {
		t.Fatalf("prompt must stay last: %v", out)
	}
}

func TestCodexUserProviderArgvKeepsWebSearchForOpenAI(t *testing.T) {
	out, _ := CodexUserProviderArgv([]string{"codex", "exec", "do it"}, "https://api.openai.com/v1")
	if strings.Contains(strings.Join(out, " "), "web_search") {
		t.Fatalf("web_search should stay available on OpenAI: %v", out)
	}
}
