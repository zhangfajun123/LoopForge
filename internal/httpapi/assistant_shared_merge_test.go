package httpapi

import (
	"testing"
	"time"

	"agentflow/internal/domain"
)

// A reader that asks the next question before the previous reply reached its
// transcript used to be rejected with 「共享会话已更新，请重新打开后发送」. The
// server already has the reply, so the merge keeps it and appends the question.
func TestMergeAssistantTranscriptToleratesMissingReply(t *testing.T) {
	reply := domain.AssistantMessage{Role: "assistant", Content: "方案设计节点用 generic-docker"}
	server := []assistantMessage{
		{Role: "user", Content: "第一个问题"},
		reply,
	}
	client := []assistantMessage{
		{Role: "user", Content: "第一个问题"},
		{Role: "user", Content: "我想改成远端的 worker 在哪里指定？"},
	}
	merged, forged, ok := mergeAssistantTranscript(server, client)
	_ = forged
	if !ok {
		t.Fatal("merge rejected a client that only lags on the reply")
	}
	if len(merged) != 3 {
		t.Fatalf("merged transcript = %d messages, want 3", len(merged))
	}
	if merged[0].Content != "第一个问题" || merged[1].Content != reply.Content || merged[2].Content != "我想改成远端的 worker 在哪里指定？" {
		t.Fatalf("unexpected order: %+v", merged)
	}
}

func TestMergeAssistantTranscriptKeepsPlainPrefix(t *testing.T) {
	server := []assistantMessage{{Role: "user", Content: "a"}}
	client := []assistantMessage{{Role: "user", Content: "a"}, {Role: "user", Content: "b"}}
	merged, forged, ok := mergeAssistantTranscript(server, client)
	_ = forged
	if !ok || len(merged) != 2 || merged[1].Content != "b" {
		t.Fatalf("plain append failed: ok=%v merged=%+v", ok, merged)
	}
}

// A stored turn carries server-side metadata the browser transcript never has.
// Treating that as "a different message" made every shared conversation answer
// 409 from its first turn, because the client's very first user message could
// never match the stored copy.
func TestMergeAssistantTranscriptMatchesStoredTurnWithoutClientMetadata(t *testing.T) {
	now := time.Now().UTC()
	server := []assistantMessage{{
		Role: "user", Content: "在当前画布上的这份 SOP 上按要求修改",
		AuthorID: 7, AuthorName: "owner", CreatedAt: now,
	}}
	client := []assistantMessage{{Role: "user", Content: "在当前画布上的这份 SOP 上按要求修改"}}
	merged, _, ok := mergeAssistantTranscript(server, client)
	if !ok {
		t.Fatal("merge rejected a client whose turn is already stored")
	}
	if len(merged) != 1 {
		t.Fatalf("the stored turn was appended again: %+v", merged)
	}
	if merged[0].AuthorID != 7 || !merged[0].CreatedAt.Equal(now) {
		t.Fatalf("stored attribution was lost: %+v", merged[0])
	}
}

// Two readers racing: the later one holds a transcript whose user message is
// already stored as well.
func TestMergeAssistantTranscriptIgnoresAlreadyStoredMessage(t *testing.T) {
	now := time.Now().UTC()
	server := []assistantMessage{{Role: "user", Content: "同一个问题", CreatedAt: now}}
	client := []assistantMessage{{Role: "user", Content: "同一个问题", CreatedAt: now}}
	merged, forged, ok := mergeAssistantTranscript(server, client)
	_ = forged
	if !ok || len(merged) != 1 {
		t.Fatalf("duplicate question was appended again: ok=%v merged=%+v", ok, merged)
	}
}

// The orchestrate panel echoes the reply it rendered as prose while the server
// keeps the JSON draft. That reply copy must not break the turn, and it must
// never reach storage either.
func TestMergeAssistantTranscriptIgnoresClientReplyCopy(t *testing.T) {
	server := []assistantMessage{
		{Role: "user", Content: "编排请求"},
		{Role: "assistant", Content: `{"drafts":[{"tool":"workflows_save","body":{}}]}`},
	}
	client := []assistantMessage{
		{Role: "user", Content: "编排请求"},
		{Role: "assistant", Content: "已核对当前 SOP，改动计划如下…"}, // reply.message, not the stored JSON
		{Role: "user", Content: "再改一个节点"},
	}
	merged, forged, ok := mergeAssistantTranscript(server, client)
	if forged {
		t.Fatal("the client's own reply rendering was treated as a forgery")
	}
	if !ok {
		t.Fatal("merge rejected a client that echoes its own rendered reply")
	}
	if len(merged) != 3 {
		t.Fatalf("merged transcript = %d messages, want 3: %+v", len(merged), merged)
	}
	if merged[1].Role != "assistant" || merged[1].Content != `{"drafts":[{"tool":"workflows_save","body":{}}]}` {
		t.Fatalf("stored reply was replaced by the client's prose: %+v", merged[1])
	}
	if merged[2].Content != "再改一个节点" {
		t.Fatalf("new question not appended: %+v", merged)
	}
}

// The panel keeps only its last 8 messages, so a long conversation arrives as a
// suffix of the stored transcript. That used to answer 409 on every send.
func TestMergeAssistantTranscriptAcceptsTruncatedHistory(t *testing.T) {
	server := []assistantMessage{
		{Role: "user", Content: "第一轮问题"},
		{Role: "assistant", Content: `{"drafts":[]}`},
		{Role: "user", Content: "第二轮问题"},
		{Role: "assistant", Content: `{"drafts":[]}`},
	}
	client := []assistantMessage{ // the leading two messages were trimmed away
		{Role: "user", Content: "第二轮问题"},
		{Role: "assistant", Content: "第二轮的口播摘要"},
		{Role: "user", Content: "第三轮问题"},
	}
	merged, _, ok := mergeAssistantTranscript(server, client)
	if !ok {
		t.Fatal("a suffix-only transcript was rejected")
	}
	if len(merged) != 5 || merged[4].Content != "第三轮问题" {
		t.Fatalf("suffix transcript merged wrong: %+v", merged)
	}
	if merged[0].Content != "第一轮问题" || merged[2].Content != "第二轮问题" {
		t.Fatalf("stored history was rewritten: %+v", merged)
	}
}

// A client may not rewrite a stored turn, but it also must not be blocked: an
// edited question is kept as the stored text plus the new turn, append-only.
func TestMergeAssistantTranscriptKeepsStoredTurnOnEditedQuestion(t *testing.T) {
	server := []assistantMessage{
		{Role: "user", Content: "原问题"},
		{Role: "assistant", Content: "回答"},
	}
	client := []assistantMessage{
		{Role: "user", Content: "被改写的问题"},
	}
	merged, _, ok := mergeAssistantTranscript(server, client)
	if !ok {
		t.Fatal("an edited question deadlocked the conversation")
	}
	if len(merged) != 3 {
		t.Fatalf("merged transcript = %d messages, want 3: %+v", len(merged), merged)
	}
	if merged[0].Content != "原问题" {
		t.Fatalf("stored question was rewritten: %+v", merged[0])
	}
	if merged[2].Content != "被改写的问题" || merged[2].Role != "user" {
		t.Fatalf("edited question was not appended as a new turn: %+v", merged[2])
	}
}

// Repeating identical text later in the same conversation is a new turn, not a
// replay: the cursor only moves forward.
func TestMergeAssistantTranscriptAppendsRepeatedQuestion(t *testing.T) {
	server := []assistantMessage{
		{Role: "user", Content: "继续"},
		{Role: "assistant", Content: `{"drafts":[]}`},
	}
	client := []assistantMessage{
		{Role: "user", Content: "继续"},
		{Role: "user", Content: "继续"},
	}
	merged, _, ok := mergeAssistantTranscript(server, client)
	if !ok || len(merged) != 3 {
		t.Fatalf("repeated question should append once: ok=%v merged=%+v", ok, merged)
	}
}