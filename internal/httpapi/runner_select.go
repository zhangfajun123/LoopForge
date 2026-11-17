package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"agentflow/internal/domain"
)

// executionRunnerPick resolves a per-run 执行机器 (execution machine) chosen in
// the UI. An empty pick keeps today's behaviour: the responsible user's default
// machine, then the cloud pool, then this host.
//
// A runner receives the responsible user's credentials and code, so the pick is
// authorised, not merely validated:
//   - the caller's own machine is always allowed;
//   - a platform machine (owner 0) is allowed when the cloud pool reaches it for
//     this project;
//   - another person's machine is refused;
//   - admins may pick any machine, which is how a run is moved off a machine
//     that is going away.
//
// The returned id is what the run's execution context freezes.
func (d *deps) executionRunnerPick(callerID int64, projectID, runnerID string) (string, error) {
	runnerID = strings.TrimSpace(runnerID)
	if runnerID == "" {
		return "", nil
	}
	if callerID <= 0 {
		return "", HTTPError{Status: http.StatusUnauthorized, Message: "选择执行机器需要先登录"}
	}
	runner, ok := d.repo.Runner(runnerID)
	if !ok {
		return "", HTTPError{Status: http.StatusBadRequest, Message: "执行机器不存在，请刷新后重选"}
	}
	name := runnerPickName(runner)
	if runner.OwnerUserID != callerID {
		allowed := runner.OwnerUserID == 0 && d.engine != nil && d.engine.PoolEligibleRunner(runnerID, projectID)
		if !allowed {
			caller, found := d.repo.User(callerID)
			if !found || caller.PlatformRole != domain.PlatformRoleAdmin {
				if runner.OwnerUserID != 0 {
					return "", HTTPError{Status: http.StatusForbidden, Message: fmt.Sprintf("不能使用他人的执行机器 %s", name)}
				}
				return "", HTTPError{Status: http.StatusForbidden, Message: fmt.Sprintf("执行机器 %s 不在当前项目的云端执行池内", name)}
			}
		}
	}
	if runner.Status != domain.RunnerStatusOnline {
		return "", HTTPError{Status: http.StatusConflict, Message: fmt.Sprintf("执行机器 %s 当前不在线，请等它上线或选择其他机器", name)}
	}
	if !runner.SupportsReportedOutputs() || !runner.SupportsIntegrate() {
		return "", HTTPError{Status: http.StatusConflict, Message: fmt.Sprintf("执行机器 %s 的 Runner 版本是 %s，不支持远端执行，请先升级", name, strings.TrimSpace(runner.RunnerVersion))}
	}
	return runner.ID, nil
}

// runnerPickName is the label an operator recognises a machine by.
func runnerPickName(runner domain.Runner) string {
	if name := strings.TrimSpace(runner.Name); name != "" {
		return name
	}
	return runner.ID
}