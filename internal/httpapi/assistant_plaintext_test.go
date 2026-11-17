package httpapi

import (
	"strings"
	"testing"

	"agentflow/internal/runtime"
)

// Hermes answers in plain text: no type=result envelope ever reaches the
// protocol decoder, which used to fail the turn with MISSING_RESULT (phase
// protocol) even though the CLI exited 0 with a complete answer.
func TestPlainTextCLIAnswerBecomesTheResult(t *testing.T) {
	translate := runtime.NewAgentCLITranslator(runtime.AgentCLIHermes)
	if translate == nil {
		t.Fatal("hermes translator is nil")
	}
	var answer strings.Builder
	for _, raw := range []string{"```", "ok", "```"} {
		for _, line := range translate(raw) {
			if text := assistantTextDelta(line); text != "" {
				answer.WriteString(text)
			}
		}
	}
	if got := strings.TrimSpace(answer.String()); !strings.Contains(got, "ok") {
		t.Fatalf("accumulated answer = %q, want it to contain ok", got)
	}
	if out, err := decodeAssistantCLIResult(answer.String(), false, nil, nil); err != nil || !strings.Contains(out, "ok") {
		t.Fatalf("plain text answer must decode as the result: out=%q err=%v", out, err)
	}
}

func TestResultEnvelopeIsNotMistakenForAPlainTextDelta(t *testing.T) {
	if text := assistantTextDelta(`{"type":"result","subtype":"success","result":"done"}`); text != "" {
		t.Fatalf("result envelope yielded a delta %q", text)
	}
	if text := assistantTextDelta("not json"); text != "" {
		t.Fatalf("non-JSON line yielded a delta %q", text)
	}
}