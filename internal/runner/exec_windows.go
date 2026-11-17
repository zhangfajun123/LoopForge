//go:build windows

package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	flowruntime "agentflow/internal/runtime"
)

// setProcessGroup isolates the CLI in a new process group so a cancel can end
// the CLI together with the wrappers and MCP servers it started.
func setProcessGroup(cmd *exec.Cmd) {
	flowruntime.ConfigureProcessGroup(cmd)
}

// terminateProcessGroup ends the CLI process tree. Windows has no graceful
// signal for an arbitrary console process, so the taskkill/T fallback kills the
// direct child and its descendants immediately.
func terminateProcessGroup(cmd *exec.Cmd, processDone <-chan struct{}) {
	select {
	case <-processDone:
		return
	default:
	}
	if cmd.Process == nil {
		return
	}
	flowruntime.SignalProcessGroup(cmd, true)
}

// platformCommandContext builds the direct child process. npm installs Agent
// CLIs as `.cmd` shims on Windows, and CreateProcess cannot launch those
// directly, so they run through cmd.exe. Executables (.exe, .com) run without a
// wrapper.
func platformCommandContext(ctx context.Context, argv []string) *exec.Cmd {
	if len(argv) == 0 {
		return exec.CommandContext(ctx, "")
	}
	switch strings.ToLower(filepath.Ext(argv[0])) {
	case ".cmd", ".bat":
		line := quoteForCmd(argv[0])
		for _, arg := range argv[1:] {
			line += " " + quoteForCmd(arg)
		}
		return exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", line)
	}
	return exec.CommandContext(ctx, argv[0], argv[1:]...)
}

// quoteForCmd quotes one cmd.exe argument. The characters that would otherwise
// break cmd's own parsing are neutralized by double quotes; embedded quotes are
// escaped as \" for the CreateProcess command line.
func quoteForCmd(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, ` \t"&|<>^()%!`) {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

// lookupExecutableInDir resolves name inside dir. Windows has no executable
// bit; a file is runnable when its extension is in PATHEXT (".COM;.EXE;.BAT;
// .CMD;..." by default). npm/vendor shims are .cmd files, so bare names are
// probed with each extension.
func lookupExecutableInDir(dir, name string) (string, bool) {
	extensions := []string{""}
	if filepath.Ext(name) == "" {
		extensions = windowsExecutableExtensions()
	}
	for _, extension := range extensions {
		candidate := filepath.Join(dir, name+extension)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Size() == 0 {
			continue
		}
		return candidate, true
	}
	return "", false
}

func windowsExecutableExtensions() []string {
	raw := strings.TrimSpace(os.Getenv("PATHEXT"))
	if raw == "" {
		raw = ".COM;.EXE;.BAT;.CMD"
	}
	extensions := make([]string, 0, len(raw)/4+1)
	for _, part := range strings.Split(raw, ";") {
		if part = strings.TrimSpace(part); part != "" {
			extensions = append(extensions, strings.ToLower(part))
		}
	}
	return extensions
}

// platformExtraSearchPaths lists directories a Windows installer commonly puts
// Agent CLIs in but a service account's PATH omits.
func platformExtraSearchPaths(home string) []string {
	paths := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
		filepath.Join(home, "AppData", "Roaming", "npm"),
		filepath.Join(home, "AppData", "Local", "Programs"),
	}
	if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
		paths = append(paths, filepath.Join(appData, "npm"))
	}
	if programFiles := strings.TrimSpace(os.Getenv("ProgramFiles")); programFiles != "" {
		paths = append(paths, filepath.Join(programFiles, "nodejs"))
	}
	return paths
}
