package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// updateProcess returns a process of ws v1.0.0 in /h/.local/bin, with v1.1.0 the latest release.
func updateProcess(t *testing.T) (cli.Env, *effecttest.Machine) {
	t.Helper()
	body := []byte("new ws")
	sum := sha256.Sum256(body)
	files := map[string][]byte{
		"/download/v1.1.0/ws.sha256":       []byte(hex.EncodeToString(sum[:]) + "  ws-darwin-arm64\n"),
		"/download/v1.1.0/ws-darwin-arm64": body,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/v1.1.0", http.StatusFound)
			return
		}
		_, _ = w.Write(files[r.URL.Path])
	}))
	t.Cleanup(srv.Close)
	p := clitest.Env()
	m := p.Machine.(*effecttest.Machine)
	m.Files["/h/.local/bin/ws"] = []byte("old ws")
	m.Results["/h/.local/bin/.ws.7 version"] = effecttest.Result{Out: []byte("v1.1.0\n")}
	m.Results["rm -f /h/.local/bin/.ws.7"] = effecttest.Result{}
	p.Version, p.Releases = "v1.0.0", srv.URL
	return p, m
}

func TestUpdate(t *testing.T) {
	p, m := updateProcess(t)
	status, stdout, stderr := runWs(p, "update")
	if status != 0 || stdout != "Updated ws v1.0.0 to v1.1.0: /h/.local/bin/ws\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if got := string(m.Files["/h/.local/bin/ws"]); got != "new ws" {
		t.Errorf("the file ws runs from holds %q, want the release's", got)
	}
}

func TestUpdateAtTheLatest(t *testing.T) {
	p, m := updateProcess(t)
	p.Version = "v1.1.0"
	status, stdout, stderr := runWs(p, "update")
	if status != 0 || stdout != "ws v1.1.0 is the latest release\n" || stderr != "" || len(m.Calls) != 0 {
		t.Errorf("status = %d, stdout = %q, stderr = %q, calls = %v", status, stdout, stderr, m.Calls)
	}
}

func TestUpdateRefusesABuild(t *testing.T) {
	p, m := updateProcess(t)
	p.Version = "devel"
	status, stdout, stderr := runWs(p, "update")
	if status != 1 || stdout != "" || !strings.HasPrefix(stderr, "ws update: this ws is devel, not a release") || len(m.Calls) != 0 {
		t.Errorf("status = %d, stdout = %q, stderr = %q, calls = %v", status, stdout, stderr, m.Calls)
	}
}

func TestUpdateWithoutItsFile(t *testing.T) {
	p, _ := updateProcess(t)
	p.Self = func() (string, error) { return "", errors.New("no such file") }
	status, _, stderr := runWs(p, "update")
	if status != 1 || stderr != "ws update: finding the file ws runs from: no such file\n" {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

// TestUpdateSignal sends this process SIGTERM, which ws update catches while it waits for a server.
func TestUpdateSignal(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	p, m := updateProcess(t)
	p.Releases = srv.URL
	go func() {
		<-started
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
	status, _, stderr := runWs(p, "update")
	if status != 128+int(syscall.SIGTERM) || stderr != "" || string(m.Files["/h/.local/bin/ws"]) != "old ws" {
		t.Errorf("status = %d, stderr = %q, file %q", status, stderr, m.Files["/h/.local/bin/ws"])
	}
}
