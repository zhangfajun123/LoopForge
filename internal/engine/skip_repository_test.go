package engine

import (
	"strings"
	"testing"
	"time"

	"agentflow/internal/domain"
	"agentflow/internal/repo"
	"agentflow/internal/runtime"
)

// A SOP that declares it never touches the repository must start without one,
// even when the caller passes repositories and the project has a default
// repository: that checkout is what strands a content SOP on a machine with no
// access to the code host.
func TestStartSkipsRepositoryCheckoutWhenSOPDeclaresNone(t *testing.T) {
	t.Parallel()
	s, err := repo.OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.PutProject(domain.Project{ID: "p", Name: "P"}); err != nil {
		t.Fatal(err)
	}
	repository := domain.Repository{ID: "repo-skip", ProjectID: "p", Name: "Repo", CloneURL: "https://github.com/example/repo.git"}
	if err := s.PutRepository(repository); err != nil {
		t.Fatal(err)
	}
	nodes := []domain.WorkflowNode{{ID: "work", Type: "command", Command: "true", MaxAttempts: 1}}
	skipWorkflow := domain.WorkflowDefinition{ID: "content-line", Name: "Content", SandboxProfileID: "local", SkipRepository: true, Nodes: nodes}
	codeWorkflow := domain.WorkflowDefinition{ID: "code-line", Name: "Code", SandboxProfileID: "local", Nodes: nodes}
	for _, wf := range []domain.WorkflowDefinition{skipWorkflow, codeWorkflow} {
		if err := s.PutWorkflow(wf); err != nil {
			t.Fatal(err)
		}
	}
	e := New(s, runtime.NewManager(t.TempDir()))
	e.ConfigureExecution("skip-worker", "skip-host", time.Minute, true)
	contexts := []domain.RepositoryContext{{RepositoryID: repository.ID, Name: repository.Name, CloneURL: repository.CloneURL, WorkBranch: "agentflow/work"}}

	skipped, err := e.StartWithRepositoriesContext(skipWorkflow.ID, "", contexts, domain.TelemetryContext{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped.Repositories) != 0 || skipped.Repository.RepositoryID != "" {
		t.Fatalf("content SOP still got a repository: %+v / %+v", skipped.Repositories, skipped.Repository)
	}
	var noted bool
	for _, event := range skipped.Events {
		if event.Type == "repository.skipped" {
			noted = true
		}
	}
	if !noted {
		t.Fatalf("run should record why it has no checkout, events=%+v", skipped.Events)
	}

	kept, err := e.StartWithRepositoriesContext(codeWorkflow.ID, "", contexts, domain.TelemetryContext{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.Repositories) != 1 || strings.TrimSpace(kept.Repository.CloneURL) == "" {
		t.Fatalf("code SOP must keep its repository: %+v", kept.Repositories)
	}
}