package httpapi

import (
	"strings"
	"testing"

	"agentflow/internal/domain"
	"agentflow/internal/engine"
	"agentflow/internal/repo"
	"agentflow/internal/runtime"
)

// executionRunnerPick is the single rule both start endpoints and the selector
// endpoint use. A runner receives the responsible user's credentials, so the
// matrix below is a security boundary, not a convenience check.
func TestExecutionRunnerPickAuthorisation(t *testing.T) {
	t.Parallel()
	s, err := repo.OpenTest()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	runners := []domain.Runner{
		{ID: "runner-mine", Name: "我的机器", Status: domain.RunnerStatusOnline, RunnerVersion: "0.10.1", OwnerUserID: 42},
		{ID: "runner-theirs", Name: "别人的机器", Status: domain.RunnerStatusOnline, RunnerVersion: "0.10.1", OwnerUserID: 7},
		{ID: "runner-offline", Name: "离线机器", Status: domain.RunnerStatusOffline, RunnerVersion: "0.10.1", OwnerUserID: 42},
		{ID: "runner-pool", Name: "云端池机器", Status: domain.RunnerStatusOnline, RunnerVersion: "0.10.1", Labels: []string{"lf-cloud"}},
		{ID: "runner-old", Name: "老版本机器", Status: domain.RunnerStatusOnline, RunnerVersion: "0.1.0", OwnerUserID: 42},
	}
	for _, runner := range runners {
		if err := s.PutRunner(runner); err != nil {
			t.Fatal(err)
		}
	}
	e := engine.New(s, runtime.NewManager(t.TempDir()))
	e.ConfigureCloudRunnerPool(engine.CloudRunnerPool{Label: "lf-cloud", Projects: []string{"p"}})
	d := &deps{repo: s, engine: e}

	cases := []struct {
		name     string
		caller   int64
		project  string
		runnerID string
		wantID   string
		wantErr  string
	}{
		{name: "no pick keeps today behaviour", caller: 42, project: "p"},
		{name: "anonymous cannot pick", caller: 0, project: "p", runnerID: "runner-mine", wantErr: "选择执行机器需要先登录"},
		{name: "unknown machine", caller: 42, project: "p", runnerID: "runner-gone", wantErr: "执行机器不存在"},
		{name: "own online machine", caller: 42, project: "p", runnerID: "runner-mine", wantID: "runner-mine"},
		{name: "another user machine", caller: 42, project: "p", runnerID: "runner-theirs", wantErr: "不能使用他人的执行机器"},
		{name: "own machine offline", caller: 42, project: "p", runnerID: "runner-offline", wantErr: "当前不在线"},
		{name: "pool machine for served project", caller: 42, project: "p", runnerID: "runner-pool", wantID: "runner-pool"},
		{name: "pool machine for other project", caller: 42, project: "q", runnerID: "runner-pool", wantErr: "不在当前项目的云端执行池内"},
		{name: "runner too old for remote work", caller: 42, project: "p", runnerID: "runner-old", wantErr: "不支持远端执行"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.executionRunnerPick(tc.caller, tc.project, tc.runnerID)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got none (id=%q)", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantID {
				t.Fatalf("picked %q, want %q", got, tc.wantID)
			}
		})
	}
}