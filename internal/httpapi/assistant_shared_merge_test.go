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
	merged, ok := mergeAssistantTranscript(server, client)
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
	merged, ok := mergeAssistantTranscript(server, client)
	if !ok || len(merged) != 2 || merged[1].Content != "b" {
		t.Fatalf("plain append failed: ok=%v merged=%+v", ok, merged)
	}
}

func TestMergeAssistantTranscriptRejectsRewrites(t *testing.T) {
	server := []assistantMessage{
		{Role: "user", Content: "原问题"},
		{Role: "assistant", Content: "回答"},
	}
	// The client edited a user message it had already sent.
	rewritten := []assistantMessage{
		{Role: "user", Content: "被改写的问题"},
		{Role: "assistant", Content: "回答"},
	}
	if _, ok := mergeAssistantTranscript(server, rewritten); ok {
		t.Fatal("a rewritten user message was accepted")
	}
	// The client tried to inject an assistant reply of its own.
	forged := []assistantMessage{
		{Role: "user", Content: "原问题"},
		{Role: "assistant", Content: "伪造的回答"},
	}
	if _, ok := mergeAssistantTranscript(server, forged); ok {
		t.Fatal("a forged assistant reply was accepted")
	}
}

// Two readers racing: the later one holds a transcript whose user message is
// already stored as well.
func TestMergeAssistantTranscriptIgnoresAlreadyStoredMessage(t *testing.T) {
	now := time.Now().UTC()
	server := []assistantMessage{{Role: "user", Content: "同一个问题", CreatedAt: now}}
	client := []assistantMessage{{Role: "user", Content: "同一个问题", CreatedAt: now}}
	merged, ok := mergeAssistantTranscript(server, client)
	if !ok || len(merged) != 1 {
		t.Fatalf("duplicate question was appended again: ok=%v merged=%+v", ok, merged)
	}
}