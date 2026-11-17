package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The product is LoopForge. "AgentFlow" survives only in internal identifiers
// (AgentFlowHarnessIcons, agentflow-* message types, cookies), never as a word
// a user can read in the console or chat UI.
var visibleOldBrand = regexp.MustCompile(`AgentFlow([^A-Za-z0-9_]|$)`)

// The login page is shipped from the embedded bundle, so its footer link must
// point at this project's GitHub repository.
func TestLoginPageGitHubLinkTargetsProjectRepository(t *testing.T) {
	page, err := fs.ReadFile(Files, "login.html")
	if err != nil {
		t.Fatal(err)
	}
	const want = `href="https://github.com/zhangfajun123/LoopForge"`
	if !strings.Contains(string(page), want) {
		t.Errorf("login.html is missing the project GitHub link %s", want)
	}
	if strings.Contains(string(page), "https://github.com/Tencent/LoopForge") {
		t.Error("login.html still links to the old GitHub repository")
	}
}

func TestUserFacingFrontendSaysLoopForge(t *testing.T) {
	roots := []string{"app", "extensions", "../chatbot/src", "../chatbot/public"}
	files := []string{"../chatbot/index.html", "mcp-setup.txt"}
	for _, pattern := range []string{"*.js", "*.html"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			switch filepath.Ext(path) {
			case ".js", ".jsx", ".html", ".json", ".txt":
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.js") || strings.HasSuffix(file, ".test.js") || strings.HasSuffix(file, ".test.jsx") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for number, line := range strings.Split(string(data), "\n") {
			if visibleOldBrand.MatchString(line) {
				t.Errorf("%s:%d still shows the old product name: %s", file, number+1, strings.TrimSpace(line))
			}
		}
	}
}
