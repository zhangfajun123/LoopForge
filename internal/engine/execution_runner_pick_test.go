package engine

import (
	"testing"
	"time"

	"agentflow/internal/domain"
	"agentflow/internal/repo"
	"agentflow/internal/runtime"
)

// A per-run 执行机器 pick must survive starting: the responsible user's default
// only fills the field when the caller named no machine.
func TestStartKeepsPickedExecutionRunner(t *testing.T) {
	t.Parallel()
	s, err := repo.OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"runner-default", "runner-picked"} {
		if err := s.PutRunner(domain.Runner{ID: id, Name: id, Status: domain.RunnerStatusOnline, RunnerVersion: "0.10.1"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetDefaultRunner(42, "runner-default", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	wf := domain.WorkflowDefinition{ID: "pick-machine", Name: "Pick", SandboxProfileID: "local", Nodes: []domain.WorkflowNode{{ID: "work", Type: "command", Command: "true", MaxAttempts: 1}}}
	if err := s.PutWorkflow(wf); err != nil {
		t.Fatal(err)
	}
	e := New(s, runtime.NewManager(t.TempDir()))
	e.ConfigureExecution("pick-worker", "pick-host", time.Minute, true)

	picked, err := e.StartWithRepositoriesContext(wf.ID, "", nil, domain.TelemetryContext{ProjectID: "p", ResponsibleUserID: 42, ExecutionRunnerID: "runner-picked"})
	if err != nil {
		t.Fatal(err)
	}
	if got := picked.ExecutionContext.ExecutionRunnerID; got != "runner-picked" {
		t.Fatalf("picked machine was overwritten by the default: %q", got)
	}
	if picked.ExecutionContext.ExecutionMode != domain.ExecutionModeRunner {
		t.Fatalf("execution mode = %q", picked.ExecutionContext.ExecutionMode)
	}

	fallback, err := e.StartWithRepositoriesContext(wf.ID, "", nil, domain.TelemetryContext{ProjectID: "p", ResponsibleUserID: 42})
	if err != nil {
		t.Fatal(err)
	}
	if got := fallback.ExecutionContext.ExecutionRunnerID; got != "runner-default" {
		t.Fatalf("a run without a pick should use the default machine, got %q", got)
	}
}