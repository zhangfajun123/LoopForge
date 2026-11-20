//go:build !windows

package runtime

import (
	"context"
	"os"
	"os/exec"
)

func LookPath(file string) (string, error) { return exec.LookPath(file) }
func pathNameCandidates(name string) []string { return []string{name} }
func executableFile(info os.FileInfo) bool     { return info.Mode().IsRegular() && info.Mode()&0o111 != 0 }

func NewCommand(argv ...string) *exec.Cmd {
	if len(argv) == 0 {
		return exec.Command("")
	}
	return exec.Command(argv[0], argv[1:]...)
}

func NewCommandContext(ctx context.Context, argv ...string) *exec.Cmd {
	if len(argv) == 0 {
		return exec.CommandContext(ctx, "")
	}
	return exec.CommandContext(ctx, argv[0], argv[1:]...)
}
