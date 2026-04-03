package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/imjasonh/infinite-git/internal/repo"
	"github.com/imjasonh/infinite-git/internal/server"
)

func TestGoCloneAndPull(t *testing.T) {
	modulePath := "example.com/infinite-go"
	content := newGoContent(modulePath)
	serverRepo, err := repo.New(t.TempDir(), content.InitialFiles())
	if err != nil {
		t.Fatalf("failed to create server repo: %v", err)
	}
	srv := server.New(serverRepo, content)
	ts := httptest.NewServer(goGetMiddleware(modulePath, srv.Handler()))
	defer ts.Close()

	// Clone the repository
	clientDir := t.TempDir()
	gitRepo, err := git.PlainClone(clientDir, false, &git.CloneOptions{
		URL: ts.URL,
	})
	if err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	// Verify go.mod exists with correct module path
	goMod, err := os.ReadFile(filepath.Join(clientDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(goMod), fmt.Sprintf("module %s", modulePath)) {
		t.Errorf("go.mod does not contain expected module path, got: %s", goMod)
	}

	// Verify pulltime.go exists with PullTime var
	goFile, err := os.ReadFile(filepath.Join(clientDir, "pulltime.go"))
	if err != nil {
		t.Fatalf("failed to read pulltime.go: %v", err)
	}
	goFileStr := string(goFile)
	if !strings.Contains(goFileStr, "package infinitego") {
		t.Errorf("pulltime.go does not contain expected package name, got: %s", goFile)
	}
	if !strings.Contains(goFileStr, "var PullTime = time.Date(") {
		t.Errorf("pulltime.go does not contain PullTime var, got: %s", goFile)
	}

	// Pull again and verify PullTime changed
	firstGoFile := goFileStr
	w, err := gitRepo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		t.Fatalf("pull failed: %v", err)
	}
	goFile, err = os.ReadFile(filepath.Join(clientDir, "pulltime.go"))
	if err != nil {
		t.Fatalf("failed to read pulltime.go after pull: %v", err)
	}
	if string(goFile) == firstGoFile {
		t.Error("pulltime.go did not change after pull")
	}
}

func TestGoGetDiscovery(t *testing.T) {
	modulePath := "example.com/infinite-go"
	content := newGoContent(modulePath)
	serverRepo, err := repo.New(t.TempDir(), content.InitialFiles())
	if err != nil {
		t.Fatalf("failed to create server repo: %v", err)
	}
	srv := server.New(serverRepo, content)
	ts := httptest.NewServer(goGetMiddleware(modulePath, srv.Handler()))
	defer ts.Close()

	// Test ?go-get=1 returns go-import meta tag
	resp, err := http.Get(ts.URL + "/?go-get=1")
	if err != nil {
		t.Fatalf("failed to fetch go-get: %v", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	expected := fmt.Sprintf(`<meta name="go-import" content="%s git`, modulePath)
	if !strings.Contains(body, expected) {
		t.Errorf("go-get response does not contain expected meta tag.\nexpected to contain: %s\ngot: %s", expected, body)
	}
}
