package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"agentflow/internal/domain"
	"agentflow/internal/engine"
	"agentflow/internal/repo"
	"agentflow/internal/runtime"
	"agentflow/internal/tracker"
)

// POST /api/runs must start the run on the machine the caller picked, while a
// request that names no machine still follows the responsible user's default.
// This exercises the real gin route, not the engine entry point directly.
func TestStartRunHTTPAcceptsExecutionRunnerPick(t *testing.T) {
	s, err := repo.OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.PutUser(domain.User{LoginName: "pick-owner", ChineseName: "选机负责人"}); err != nil {
		t.Fatal(err)
	}
	user, ok := s.UserByLoginName("pick-owner")
	if !ok {
		t.Fatal("user not stored")
	}
	t.Setenv("AGENTFLOW_PREVIEW_LOCAL_LOGIN", "1")
	t.Setenv("AGENTFLOW_LOCAL_LOGIN_USER", "pick-owner")

	for _, id := range []string{"runner-default", "runner-picked"} {
		if err := s.PutRunner(domain.Runner{ID: id, Name: id, Status: domain.RunnerStatusOnline, RunnerVersion: "0.10.1", OwnerUserID: user.ID}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetDefaultRunner(user.ID, "runner-default", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	wf := domain.WorkflowDefinition{ID: "pick-http", Name: "Pick HTTP", SandboxProfileID: "local", Nodes: []domain.WorkflowNode{{ID: "work", Type: "command", Command: "true", MaxAttempts: 1}}}
	if err := s.PutWorkflow(wf); err != nil {
		t.Fatal(err)
	}
	e := engine.New(s, runtime.NewManager(filepath.Join(t.TempDir(), "workspaces")))
	e.ConfigureExecution("pick-worker", "pick-host", time.Minute, true)
	h := New(s, e, tracker.NewService(), "", "")

	start := func(body map[string]any) domain.Run {
		raw, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewReader(raw)))
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /api/runs %v = %d %s", body, rec.Code, rec.Body.String())
		}
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode run: %v", err)
		}
		run, ok := s.RunState(created.ID)
		if !ok {
			t.Fatalf("run %q not found after start", created.ID)
		}
		return run
	}

	picked := start(map[string]any{"workflowId": wf.ID, "executionRunnerId": "runner-picked"})
	if got := picked.ExecutionContext.ExecutionRunnerID; got != "runner-picked" {
		t.Fatalf("picked machine = %q, want runner-picked", got)
	}
	if picked.ExecutionContext.ExecutionMode != domain.ExecutionModeRunner {
		t.Fatalf("execution mode = %q, want runner", picked.ExecutionContext.ExecutionMode)
	}

	fallback := start(map[string]any{"workflowId": wf.ID})
	if got := fallback.ExecutionContext.ExecutionRunnerID; got != "runner-default" {
		t.Fatalf("run without a pick = %q, want the default runner", got)
	}
}