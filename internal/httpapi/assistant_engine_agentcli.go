package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agentflow/internal/runtime"
)

// assistantAgentCLIModel resolves the model a chat turn actually runs. A session
// that picked 「CLI default」 stores the literal "default", which no CLI can use
// (opencode would write provider/ "agentflow/default" and the gateway answers
// 404), so fall back to the user's saved catalog for that CLI and then to the
// built-in catalog default. An empty result means the CLI keeps its own default.
func assistantAgentCLIModel(kind, model string, cred runtime.ProviderCredential) string {
	model = strings.TrimSpace(model)
	if model != "" && !strings.EqualFold(model, "default") {
		return model
	}
	for _, saved := range cred.Models {
		if id := strings.TrimSpace(saved.ID); id != "" && !strings.EqualFold(id, "default") {
			return id
		}
	}
	if catalog, ok := executorModelCatalogs()[kind]; ok {
		return strings.TrimSpace(catalog.Default)
	}
	return ""
}

// prepareAssistantAgentCLIInvocation runs one chat turn on an agent CLI in
// its own headless mode, read-only: the session's MCP gateway stays
// available while the CLI's shell, edit and web tools are denied.
func prepareAssistantAgentCLIInvocation(kind string, o assistantEngineInvocationOpts) (assistantCLIInvocation, error) {
	o.Model = assistantAgentCLIModel(kind, o.Model, o.User)
	servers, err := readAssistantMCPServers(o.Config)
	if err != nil {
		return assistantCLIInvocation{}, err
	}
	entries := make(map[string]any, len(servers))
	for id, server := range servers {
		entry := map[string]any{}
		for key, value := range server {
			// CodeBuddy's per-tool defer_loading extension.
			if key != "tools" {
				entry[key] = value
			}
		}
		if url, _ := entry["url"].(string); strings.TrimSpace(url) != "" {
			entry["type"] = "http"
		}
		entries[id] = entry
	}
	if err := runtime.WriteAgentCLIMCPConfig(kind, runtime.AgentCLIHome{Base: o.Root, Rel: "home"}, entries, runtime.AgentCLIMCPOptions{ReadOnly: !o.Workspace}); err != nil {
		return assistantCLIInvocation{}, err
	}
	resume := ""
	if o.Resume {
		if resume = readAgentCLISession(o.Root, kind, o.SessionID); !agentCLISessionValid(resume) {
			return assistantCLIInvocation{resumeMissing: true}, nil
		}
	}
	systemFile := filepath.Join(o.Root, kind+"-system.md")
	if err := os.WriteFile(systemFile, []byte(o.SystemPrompt), 0o600); err != nil {
		return assistantCLIInvocation{}, err
	}
	agentFile := ""
	if kind == runtime.AgentCLIKimi {
		agentFile = filepath.Join(o.Root, "kimi-agent.md")
		// Kimi binds an agent at session creation. Keep its native tools stable
		// across resumes; the per-invocation PreToolUse hook gates each mode.
		if err := runtime.WriteKimiConversationAgent(agentFile, o.SystemPrompt, false); err != nil {
			return assistantCLIInvocation{}, err
		}
	}
	command := runtime.AgentCLICommand(kind, runtime.AgentCLICommandOptions{
		Binary: o.Bin, Model: o.Model, Effort: o.Effort, Resume: resume, Prompt: o.Prompt,
		SystemPrompt: o.SystemPrompt, SystemPromptFile: systemFile, Chat: true, Workspace: o.Workspace, AgentFile: agentFile,
	})
	command.CredentialBindings = []runtime.CredentialBinding{runtime.BindAgentCLI}
	translate := runtime.NewAgentCLITranslator(kind)
	native := ""
	return assistantCLIInvocation{
		command:     command,
		credentials: runtime.RunCredentials{AgentCLI: map[string]runtime.ProviderCredential{kind: o.User}},
		sessionRef:  resume,
		// Hermes answers in plain text and never emits a type=result envelope.
		plainTextAnswer: kind == runtime.AgentCLIHermes,
		translate: func(line string) []string {
			out := translate(line)
			for i, translated := range out {
				out[i] = assistantAgentCLILine(kind, translated, &native)
				if agentCLISessionValid(native) {
					_ = writeAgentCLISession(o.Root, kind, o.SessionID, native)
				}
			}
			return out
		},
		finish: func(ok bool) {
			if agentCLISessionValid(native) {
				_ = writeAgentCLISession(o.Root, kind, o.SessionID, native)
			}
		},
	}, nil
}

// assistantAgentCLILine records the native session a translated line names
// and turns a failed result into an error line: the pipeline reports error
// lines (and recognises a rejected key in them), while a failed result would
// read as an empty reply.
func assistantAgentCLILine(kind, line string, native *string) string {
	var event struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
		Result    string `json:"result"`
	}
	if json.Unmarshal([]byte(line), &event) != nil {
		return line
	}
	if id := strings.TrimSpace(event.SessionID); id != "" {
		*native = id
	}
	if event.Type != "result" || !event.IsError {
		return line
	}
	message := strings.TrimSpace(event.Result)
	if message == "" {
		message = assistantEngineLabel(kind) + " 本轮执行失败"
	}
	return codexLine(map[string]any{"type": "error", "error": message, "session_id": *native})
}

// agent-cli-sessions.json maps "<kind>:<chat session>" onto the CLI's own
// session ID, which each CLI assigns itself.
var agentCLISessionsMu sync.Mutex

// agentCLISessionValid guards the native ID before it reaches an argument.
func agentCLISessionValid(id string) bool {
	return id != "" && id[0] != '-' && codexConfigKeyRe.MatchString(id)
}

func agentCLISessionsPath(root string) string {
	return filepath.Join(root, "agent-cli-sessions.json")
}

func readAgentCLISession(root, kind, sessionID string) string {
	agentCLISessionsMu.Lock()
	defer agentCLISessionsMu.Unlock()
	sessions := map[string]string{}
	if data, err := os.ReadFile(agentCLISessionsPath(root)); err == nil {
		_ = json.Unmarshal(data, &sessions)
	}
	return sessions[kind+":"+sessionID]
}

func writeAgentCLISession(root, kind, sessionID, native string) error {
	agentCLISessionsMu.Lock()
	defer agentCLISessionsMu.Unlock()
	sessions := map[string]string{}
	if data, err := os.ReadFile(agentCLISessionsPath(root)); err == nil {
		_ = json.Unmarshal(data, &sessions)
	}
	sessions[kind+":"+sessionID] = native
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	return os.WriteFile(agentCLISessionsPath(root), data, 0o600)
}
