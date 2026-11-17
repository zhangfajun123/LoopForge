package runtime

import "os/exec"

// ConfigureProcessGroup isolates a command in its own process group so a
// cancel can reach the process and every descendant it spawned.
func ConfigureProcessGroup(cmd *exec.Cmd) {
	configureProcessGroup(cmd)
}

// SignalProcessGroup terminates a started command's process group. force
// escalates from a polite termination signal to an unconditional kill.
func SignalProcessGroup(cmd *exec.Cmd, force bool) {
	killProcessGroup(cmd, force)
}

// TerminateProcessTree force-terminates the process tree led by pid.
func TerminateProcessTree(pid int) {
	killProcessTree(pid)
}
