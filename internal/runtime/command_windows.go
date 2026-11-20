//go:build windows

package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewCommand applies the Windows launchers needed by the supported local CLI
// setup: PowerShell shims and Git for Windows shells.
func NewCommand(argv ...string) *exec.Cmd { return newWindowsCommand(nil, argv...) }

// NewCommandContext applies the same Windows launchers to a cancelable child.
func NewCommandContext(ctx context.Context, argv ...string) *exec.Cmd {
	return newWindowsCommand(ctx, argv...)
}

func newWindowsCommand(ctx context.Context, argv ...string) *exec.Cmd {
	if len(argv) == 0 {
		if ctx != nil {
			return exec.CommandContext(ctx, "")
		}
		return exec.Command("")
	}
	name, args := windowsCommandLine(argv)
	var cmd *exec.Cmd
	if ctx != nil {
		cmd = exec.CommandContext(ctx, name, args...)
	} else {
		cmd = exec.Command(name, args...)
	}
	ConfigureProcessTree(cmd)
	return cmd
}

func windowsCommandLine(argv []string) (string, []string) {
	if len(argv) == 0 {
		return "", nil
	}
	if strings.EqualFold(filepath.Ext(argv[0]), ".ps1") {
		args := []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", argv[0]}
		args = append(args, argv[1:]...)
		return "powershell.exe", args
	}
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(argv[0])), ".exe")
	if base == "sh" || base == "bash" {
		if shell := findGitBashShell(base); shell != "" {
			argv = append([]string{shell}, argv[1:]...)
		}
	}
	return argv[0], argv[1:]
}

func findGitBashShell(name string) string {
	if path, err := exec.LookPath(name + ".exe"); err == nil {
		return path
	}
	git, err := exec.LookPath("git.exe")
	if err == nil {
		for dir := filepath.Dir(git); ; dir = filepath.Dir(dir) {
			for _, candidate := range []string{
				filepath.Join(dir, "usr", "bin", name+".exe"),
				filepath.Join(dir, "bin", name+".exe"),
			} {
				if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
					return candidate
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if strings.TrimSpace(root) == "" {
			continue
		}
		candidate := filepath.Join(root, "Git", "usr", "bin", name+".exe")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
