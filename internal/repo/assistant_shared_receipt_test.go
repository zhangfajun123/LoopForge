package repo

import (
	"fmt"
	"testing"
	"time"

	"agentflow/internal/domain"
)

// The platform writes the generated title back a few seconds after the first
// turn; that bumps updated_at. A reader that still holds the pre-title snapshot
// used to be rejected with 「共享会话已更新，请重新打开」 even when it was the only
// participant, so the receipt now tolerates that case.
func TestSharedReceiptToleratesGeneratedTitle(t *testing.T) {
	st, err := OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	session, err := st.EnsureRunAssistantSession(fmt.Sprintf("receipt-%d", time.Now().UnixNano()), "receipt-project", "共享会话", 61)
	if err != nil {
		t.Fatal(err)
	}
	session.Title = "原始标题"
	session.Messages = []domain.AssistantMessage{{Role: "user", Content: "在吗"}}
	if err := st.PutAssistantSession(session.UserID, session); err != nil {
		t.Fatal(err)
	}
	snapshot, ok := st.AssistantSession(session.UserID, session.ID)
	if !ok {
		t.Fatal("session disappeared")
	}
	if !snapshot.Shared {
		t.Fatalf("expected a shared session, got %+v", snapshot)
	}

	// The platform lands the generated title after the reader opened the session.
	time.Sleep(10 * time.Millisecond)
	if err := st.SetAssistantSessionGeneratedTitle(session.UserID, session.ID, "自动生成的标题"); err != nil {
		t.Fatal(err)
	}

	receipt := snapshot
	receipt.TitleEdited = false
	if err := st.PutSharedAssistantReceipt(session.UserID, receipt, snapshot); err != nil {
		t.Fatalf("generated title blocked the receipt: %v", err)
	}
	current, _ := st.AssistantSession(session.UserID, session.ID)
	if current.Title != "自动生成的标题" {
		t.Fatalf("server title was reverted to %q", current.Title)
	}
	if !current.TitleGenerated {
		t.Fatal("generated title lost its generated flag")
	}

	// A client that edits the title on a stale snapshot is still rejected.
	stale := snapshot
	stale.Title = "用户自己改的标题"
	if err := st.PutSharedAssistantReceipt(session.UserID, stale, snapshot); err == nil {
		t.Fatal("stale title edit was accepted")
	}
}

// Another participant appending a message is a real concurrent change: the
// reader's snapshot no longer matches the server and the write must fail.
func TestSharedReceiptRejectsDivergedMessages(t *testing.T) {
	st, err := OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	session, err := st.EnsureRunAssistantSession(fmt.Sprintf("receipt-diverged-%d", time.Now().UnixNano()), "receipt-project", "共享会话", 62)
	if err != nil {
		t.Fatal(err)
	}
	session.Messages = []domain.AssistantMessage{{Role: "user", Content: "原始"}}
	if err := st.PutAssistantSession(session.UserID, session); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := st.AssistantSession(session.UserID, session.ID)

	// Someone else appends a message while our reader holds the old snapshot.
	server := snapshot
	server.Messages = []domain.AssistantMessage{
		{Role: "user", Content: "原始"},
		{Role: "user", Content: "别人也发了一句"},
	}
	if err := st.PutAssistantSession(session.UserID, server); err != nil {
		t.Fatal(err)
	}

	if err := st.PutSharedAssistantReceipt(session.UserID, snapshot, snapshot); err == nil {
		t.Fatal("diverged messages were accepted")
	}
}