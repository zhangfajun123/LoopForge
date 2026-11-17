package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agentflow/internal/domain"
)

// traceAgentCLI appends one line describing what this machine decided about an
// agent CLI invocation, to a file the operator can read.
//
// Neither of the obvious channels carries it: the node log is the platform's own
// argv (task events are keyed by (task_id, seq) and runner-chosen seqs are
// dropped), and a user systemd unit on these machines does not forward stdout to
// the journal. A file under the runner's own state directory is honest about
// both the decision and why it was taken, and costs one append per agent task.
func (d *Daemon) traceAgentCLI(task domain.RunnerTask, format string, args ...any) {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return
	}
	dir := filepath.Join(home, ".agentflow-runner")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "agentcli-decisions.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s task=%s %s\n", time.Now().UTC().Format(time.RFC3339), task.ID, fmt.Sprintf(format, args...))
}

// firstArg is argv[0] when there is one, for a trace line that must not panic.
func firstArg(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	return argv[0]
}