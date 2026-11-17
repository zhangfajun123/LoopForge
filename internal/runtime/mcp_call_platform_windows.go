//go:build windows

package runtime

import (
	"os"
	"strings"
)

func mcpCallHelperName() string {
	return "agentflow-helper.exe"
}

func mcpCallWrapperName() string {
	return "agentflow.cmd"
}

// mcpCallWrapperScript is the workspace `agentflow` shim a script node calls
// for `agentflow mcp call`. cmd.exe does not understand shebangs, so the shim is
// a batch file that forwards the arguments to the copied helper, else to the
// installed runner binary.
func mcpCallWrapperScript(helperPath, fallback string) string {
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("setlocal\r\n")
	if strings.TrimSpace(helperPath) != "" {
		b.WriteString(`if exist "` + helperPath + `" goto helper` + "\r\n")
	}
	if strings.TrimSpace(fallback) != "" {
		b.WriteString(`if exist "` + fallback + `" goto fallback` + "\r\n")
	}
	b.WriteString(`echo agentflow: mcp helper is not available in this sandbox 1>&2` + "\r\n")
	b.WriteString("exit /b 127\r\n")
	b.WriteString(":helper\r\n")
	b.WriteString(`"` + helperPath + `" %*` + "\r\n")
	b.WriteString("exit /b %ERRORLEVEL%\r\n")
	b.WriteString(":fallback\r\n")
	b.WriteString(`"` + fallback + `" %*` + "\r\n")
	b.WriteString("exit /b %ERRORLEVEL%\r\n")
	return b.String()
}

// mcpCallPathOverlay prepends the workspace helper directory so a PowerShell
// script finds `agentflow.cmd` through PATHEXT.
func mcpCallPathOverlay(environment map[string]string, hostBin string) {
	if strings.TrimSpace(hostBin) == "" {
		return
	}
	current := strings.TrimSpace(os.Getenv("PATH"))
	if current == "" {
		environment["PATH"] = hostBin
		return
	}
	environment["PATH"] = hostBin + string(os.PathListSeparator) + current
}
