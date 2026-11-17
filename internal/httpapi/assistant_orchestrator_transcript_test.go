package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"agentflow/internal/domain"
)

// The SOP orchestration panel used to deadlock every project conversation it
// opened: it echoes back the reply it rendered (`reply.message`, prose) while
// the server stored the JSON draft, and it keeps only its last 8 messages, so a
// long conversation arrives as a suffix of the stored transcript. Both shapes
// answered 409 「共享会话已更新，请重新打开后发送」 on every single send, and
// refreshing never changed the stored copy — the only escape was a new
// conversation, losing the thread. These are the two shapes, end to end.
func TestSharedConversationAcceptsOrchestratorTranscriptShapes(t *testing.T) {
	h, first, second, _, _ := sharedWorkbenchFixture(t)
	now := time.Now().UTC()
	harness := "在当前画布上的这份 SOP 上按要求修改，重新提交完整的 workflows_save 草稿。这是增量修改：用户没点名的节点、连线、id 和脚本都保持原样。"
	stored := domain.AssistantSession{
		ID:        "chat-orchestrate",
		ProjectID: "workbench",
		Title:     "编排对话",
		Messages: []domain.AssistantMessage{
			{Role: "user", Content: harness, AuthorID: first.ID, AuthorName: first.LoginName, CreatedAt: now},
			{Role: "assistant", Content: `{"message":"已核对当前 SOP","drafts":[{"tool":"workflows_save","body":{}}]}`, CreatedAt: now},
		},
	}
	if err := h.repo.PutAssistantSession(first.ID, stored); err != nil {
		t.Fatal(err)
	}
	h.generate = func(context.Context, string, domain.User) (string, error) {
		return `{"message":"第二答","drafts":[]}`, nil
	}
	send := func(messages []assistantMessage) int {
		body, _ := json.Marshal(assistantRequest{ProjectID: "workbench", SessionID: stored.ID, Messages: messages})
		c, w := workbenchContext(second, "POST", "/api/assistant/chat", bytes.NewBuffer(body))
		h.chat(c)
		return w.Code
	}

	// Shape 1: the client replays its own prose rendering of the reply.
	echoed := []assistantMessage{
		{Role: "user", Content: harness},
		{Role: "assistant", Content: "已核对当前 SOP，改动计划如下…"},
		{Role: "user", Content: harness + "\n目标：三个选题要渲染出来，让用户可见、可选"},
	}
	if code := send(echoed); code != 200 {
		t.Fatalf("echoed reply copy still blocks the turn: %d", code)
	}
	saved, _ := h.repo.AssistantSession(first.ID, stored.ID)
	if len(saved.Messages) != 4 {
		t.Fatalf("stored transcript = %d messages, want 4: %+v", len(saved.Messages), saved.Messages)
	}
	if saved.Messages[1].Content != stored.Messages[1].Content {
		t.Fatalf("the stored reply was replaced by the client's prose: %q", saved.Messages[1].Content)
	}
	if saved.Messages[2].Content != echoed[2].Content || saved.Messages[2].AuthorID != second.ID {
		t.Fatalf("new turn not appended with attribution: %+v", saved.Messages[2])
	}

	// Shape 2: the panel trimmed its history, so the transcript no longer
	// starts at the stored beginning.
	trimmed := []assistantMessage{{Role: "user", Content: "把闸口①的产物改成 HTML 卡片"}}
	if code := send(trimmed); code != 200 {
		t.Fatalf("a suffix-only transcript still blocks the turn: %d", code)
	}
	saved, _ = h.repo.AssistantSession(first.ID, stored.ID)
	if len(saved.Messages) != 6 || saved.Messages[4].Content != trimmed[0].Content {
		t.Fatalf("suffix transcript merged wrong: %+v", saved.Messages)
	}
	if saved.Messages[0].Content != harness {
		t.Fatal("the stored opening turn was rewritten")
	}
}