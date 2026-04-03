package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imjasonh/infinite-git/internal/repo"
	"github.com/imjasonh/infinite-git/internal/server"
)

func newGoTestServer(t *testing.T, modulePath string) *httptest.Server {
	t.Helper()
	content := newGoContent(modulePath)
	serverRepo, err := repo.New(t.TempDir(), content.InitialFiles())
	if err != nil {
		t.Fatalf("failed to create server repo: %v", err)
	}
	srv := server.New(serverRepo, content)
	ts := httptest.NewServer(goGetMiddleware(modulePath, srv.Handler()))
	t.Cleanup(ts.Close)
	return ts
}

// TestGoGet clones the infinite-go server repeatedly using git (which is what
// `go get` does under the hood with GOPROXY=direct) and verifies that each
// clone produces a valid, buildable Go package with a unique PullTime.
func TestGoGet(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not found in PATH")
	}
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found in PATH")
	}

	modulePath := "example.com/infinite-go"
	ts := newGoTestServer(t, modulePath)

	var pullTimes []string

	for i := 0; i < 3; i++ {
		cloneDir := t.TempDir()

		// Clone via git — this is exactly what `go get` with GOPROXY=direct does.
		cmd := exec.Command(gitBin, "clone", ts.URL, cloneDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git clone #%d failed: %v\noutput: %s", i+1, err, out)
		}

		// Verify go.mod has the right module path.
		goMod, err := os.ReadFile(filepath.Join(cloneDir, "go.mod"))
		if err != nil {
			t.Fatalf("clone #%d: failed to read go.mod: %v", i+1, err)
		}
		if !strings.Contains(string(goMod), fmt.Sprintf("module %s", modulePath)) {
			t.Fatalf("clone #%d: go.mod has wrong module path: %s", i+1, goMod)
		}

		// Verify the package builds.
		cmd = exec.Command(goBin, "build", "./...")
		cmd.Dir = cloneDir
		out, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("clone #%d: go build failed: %v\noutput: %s", i+1, err, out)
		}

		// Read pulltime.go and extract the PullTime declaration.
		data, err := os.ReadFile(filepath.Join(cloneDir, "pulltime.go"))
		if err != nil {
			t.Fatalf("clone #%d: failed to read pulltime.go: %v", i+1, err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "var PullTime") {
				pullTimes = append(pullTimes, strings.TrimSpace(line))
				t.Logf("clone #%d: %s", i+1, strings.TrimSpace(line))
				break
			}
		}
	}

	if len(pullTimes) != 3 {
		t.Fatalf("expected 3 PullTime values, got %d", len(pullTimes))
	}

	// Every clone must produce a unique PullTime.
	seen := make(map[string]bool)
	for i, pt := range pullTimes {
		if seen[pt] {
			t.Errorf("clone #%d returned duplicate PullTime: %s", i+1, pt)
		}
		seen[pt] = true
	}
}

// TestGoGetPull verifies that git pull on an already-cloned repo also
// produces a new PullTime.
func TestGoGetPull(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not found in PATH")
	}
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found in PATH")
	}

	modulePath := "example.com/infinite-go"
	ts := newGoTestServer(t, modulePath)

	cloneDir := t.TempDir()

	// Initial clone.
	cmd := exec.Command(gitBin, "clone", ts.URL, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\noutput: %s", err, out)
	}

	readPullTime := func() string {
		data, err := os.ReadFile(filepath.Join(cloneDir, "pulltime.go"))
		if err != nil {
			t.Fatalf("failed to read pulltime.go: %v", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "var PullTime") {
				return strings.TrimSpace(line)
			}
		}
		t.Fatal("pulltime.go does not contain PullTime declaration")
		return ""
	}

	prev := readPullTime()
	t.Logf("clone: %s", prev)

	for i := 0; i < 3; i++ {
		cmd := exec.Command(gitBin, "-C", cloneDir, "pull")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git pull #%d failed: %v\noutput: %s", i+1, err, out)
		}

		// Verify it still builds.
		cmd = exec.Command(goBin, "build", "./...")
		cmd.Dir = cloneDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("pull #%d: go build failed: %v\noutput: %s", i+1, err, out)
		}

		cur := readPullTime()
		t.Logf("pull #%d: %s", i+1, cur)
		if cur == prev {
			t.Errorf("pull #%d did not change PullTime", i+1)
		}
		prev = cur
	}
}

func TestGoGetDiscovery(t *testing.T) {
	modulePath := "example.com/infinite-go"
	ts := newGoTestServer(t, modulePath)

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
		t.Errorf("go-get response missing expected meta tag.\nwant substring: %s\ngot: %s", expected, body)
	}
}
