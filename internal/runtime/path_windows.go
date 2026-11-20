//go:build windows

package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LookPath also recognizes PowerShell script shims such as codex.ps1, which
// are common for per-user Node.js CLI installations on Windows.
func LookPath(file string) (string, error) {
	if path, err := exec.LookPath(file); err == nil {
		// Node CLIs commonly install both a .cmd shim and a .ps1 shim. Go's
		// Windows PATH lookup selects the .cmd file, which CreateProcess cannot
		// launch directly; prefer the PowerShell shim when both are present.
		if filepath.Ext(file) == "" && (strings.EqualFold(filepath.Ext(path), ".cmd") || strings.EqualFold(filepath.Ext(path), ".bat")) {
			if script, scriptErr := lookPathPowerShell(file); scriptErr == nil {
				return script, nil
			}
		}
		return path, nil
	}
	if strings.EqualFold(filepath.Ext(file), ".ps1") {
		return lookPathPowerShell(file)
	}
	if filepath.Ext(file) != "" {
		return "", exec.ErrNotFound
	}
	return lookPathPowerShell(file)
}

func lookPathPowerShell(file string) (string, error) {
	if strings.EqualFold(filepath.Ext(file), ".ps1") {
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			return file, nil
		}
		if !strings.ContainsAny(file, `/\\`) {
			for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
				candidate := filepath.Join(dir, file)
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					return candidate, nil
				}
			}
		}
		return "", exec.ErrNotFound
	}
	if strings.ContainsAny(file, `/\\`) {
		candidate := file + ".ps1"
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		return "", exec.ErrNotFound
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, file+".ps1")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", exec.ErrNotFound
}

func pathNameCandidates(name string) []string {
	if filepath.Ext(name) != "" {
		return []string{name}
	}
	extensions := strings.Split(os.Getenv("PATHEXT"), ";")
	if len(extensions) == 0 || strings.TrimSpace(extensions[0]) == "" {
		extensions = []string{".COM", ".EXE", ".BAT", ".CMD", ".PS1"}
	}
	out := []string{name}
	for _, extension := range extensions {
		extension = strings.TrimSpace(extension)
		if extension != "" {
			out = append(out, name+strings.ToLower(extension))
		}
	}
	if !strings.HasSuffix(strings.ToLower(strings.Join(out, "|")), ".ps1") {
		out = append(out, name+".ps1")
	}
	return out
}

func executableFile(info os.FileInfo) bool { return info.Mode().IsRegular() }
