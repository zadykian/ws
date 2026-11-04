package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// updateProcess returns a process of ws v1.0.0 in /h/.local/bin, with v1.1.0 the latest release.
func updateProcess(t *testing.T) (process, *effecttest.Machine) {
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
	p := fakeProcess()
	m := p.machine.(*effecttest.Machine)
	m.Files["/h/.local/bin/ws"] = []byte("old ws")
	m.Results["/h/.local/bin/.ws.7 version"] = effecttest.Result{Out: []byte("v1.1.0\n")}
	m.Results["rm -f /h/.local/bin/.ws.7"] = effecttest.Result{}
	p.version, p.releases = "v1.0.0", srv.URL
	return p, m
}

func TestUpdate(t *testing.T) {
	p, m := updateProcess(t)
	status, stdout, stderr := runWs(p, "", "update")
	if status != 0 || stdout != "Updated ws v1.0.0 to v1.1.0: /h/.local/bin/ws\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if got := string(m.Files["/h/.local/bin/ws"]); got != "new ws" {
		t.Errorf("the file ws runs from holds %q, want the release's", got)
	}
}

func TestUpdateAtTheLatest(t *testing.T) {
	p, m := updateProcess(t)
	p.version = "v1.1.0"
	status, stdout, stderr := runWs(p, "", "update")
	if status != 0 || stdout != "ws v1.1.0 is the latest release\n" || stderr != "" || len(m.Calls) != 0 {
		t.Errorf("status = %d, stdout = %q, stderr = %q, calls = %v", status, stdout, stderr, m.Calls)
	}
}

func TestUpdateRefusesABuild(t *testing.T) {
	p, m := updateProcess(t)
	p.version = "devel"
	status, stdout, stderr := runWs(p, "", "update")
	if status != 1 || stdout != "" || !strings.HasPrefix(stderr, "ws update: this ws is devel, not a release") || len(m.Calls) != 0 {
		t.Errorf("status = %d, stdout = %q, stderr = %q, calls = %v", status, stdout, stderr, m.Calls)
	}
}

func TestUpdateWithoutItsFile(t *testing.T) {
	p, _ := updateProcess(t)
	p.self = func() (string, error) { return "", errors.New("no such file") }
	status, _, stderr := runWs(p, "", "update")
	if status != 1 || stderr != "ws update: finding the file ws runs from: no such file\n" {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
