package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	installed = "/usr/local/bin/ws"
	copied    = "/usr/local/bin/.ws.7"
	binary    = "ws-linux-amd64"
)

// serveRelease serves release tag as the latest, with a ws of the content body, and returns its
// address. Its ws.sha256 lists the sum of body, or of listed where that is given.
func serveRelease(t *testing.T, tag, body string, listed ...string) string {
	sum := sha256.Sum256([]byte(append(listed, body)[0]))
	files := map[string]string{binary: body, "ws.sha256": hex.EncodeToString(sum[:]) + "  " + binary + "\n"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
			return
		}
		name, ok := strings.CutPrefix(r.URL.Path, "/download/"+tag+"/")
		if content, found := files[name]; ok && found {
			_, _ = w.Write([]byte(content))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// updateEnv returns the Env of a ws at version, that the releases at url serve, and its machine,
// whose installed ws is of version too.
func updateEnv(version, url string, euid int) (cli.Env, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files[installed] = []byte("old ws")
	m.Results[installed+" version"] = effecttest.Result{Out: []byte(version + "\n")}
	p := clitest.Env()
	p.Machine, p.Version, p.Releases, p.Euid = m, version, url, euid
	p.Self = func() (string, error) { return "/opt/trial/ws", nil }
	return p, m
}

func runUpdate(p cli.Env, side string, root bool, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{cli.Update(p, side, root)}, "", append([]string{"update"}, args...)...)
}

func TestUpdateRunsTheSetupOfTheWsItWrote(t *testing.T) {
	for name, tc := range map[string]struct {
		version string
		args    []string
		tag     string
	}{
		"the latest":     {"v0.9.0", nil, "v0.10.0"},
		"a trial's":      {"devel", nil, "v0.10.0"},
		"a named rc":     {"v0.9.0", []string{"--version", "0.10.0-rc.1"}, "v0.10.0-rc.1"},
		"with its v":     {"v0.9.0", []string{"--version", "v0.10.0-rc.1"}, "v0.10.0-rc.1"},
		"an older named": {"v0.11.0", []string{"--version", "0.10.0"}, "v0.10.0"},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := updateEnv(tc.version, serveRelease(t, tc.tag, "new ws"), 0)
			m.Results[copied+" version"] = effecttest.Result{Out: []byte(tc.tag + "\n")}
			status, stdout, stderr := runUpdate(p, "server", true, tc.args...)
			if status != 0 || stdout != "Updated ws "+tc.version+" to "+tc.tag+": "+installed+"\n" {
				t.Fatalf("status = %d, stdout = %q, stderr = %q; want the report", status, stdout, stderr)
			}
			if got := string(m.Files[installed]); got != "new ws" {
				t.Errorf("%s holds %q, want the release's", installed, got)
			}
			if last := m.Calls[len(m.Calls)-1]; last != "exec "+installed+" server setup" {
				t.Errorf("the last call is %q, want the server's setup of the new ws", last)
			}
		})
	}
}

func TestUpdateOfTheMacRunsItsSetup(t *testing.T) {
	p, m := updateEnv("v0.9.0", serveRelease(t, "v0.10.0", "new ws"), 501)
	m.Results[copied+" version"] = effecttest.Result{Out: []byte("v0.10.0\n")}
	if status, _, stderr := runUpdate(p, "client", false); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if last := m.Calls[len(m.Calls)-1]; last != "exec "+installed+" client setup" {
		t.Errorf("the last call is %q, want the Mac's setup", last)
	}
}

func TestUpdateOfTheReleaseAloneRunsSetup(t *testing.T) {
	// The release holds no file: a download would fail.
	p, m := updateEnv("v0.10.0-rc.1", serveRelease(t, "v0.10.0-rc.1", ""), 0)
	status, stdout, stderr := runUpdate(p, "server", true, "--version", "0.10.0-rc.1")
	if status != 0 || stdout != installed+" is ws v0.10.0-rc.1 already\n" {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if got := strings.Join(m.Calls, "\n"); got != "run "+installed+" version\nexec "+installed+" server setup" {
		t.Errorf("calls = %q, want setup of the installed ws alone", got)
	}
}

// TestUpdateBringsTheInstalledWs checks that the release of the file the units run counts, not
// that of a ws run from elsewhere. The update brings that file to the release, or refuses.
func TestUpdateBringsTheInstalledWs(t *testing.T) {
	for name, tc := range map[string]struct {
		running, installed, latest string
		status                     int
		report                     string
	}{
		"older than this ws":        {"v0.10.0", "v0.9.0", "v0.10.0", 0, "Updated ws v0.9.0 to v0.10.0: " + installed},
		"missing":                   {"v0.10.0", "", "v0.10.0", 0, "Installed ws v0.10.0: " + installed},
		"later than the latest":     {"v0.9.0", "v0.11.0", "v0.10.0", 1, installed + " is v0.11.0, later than the latest"},
		"this ws is later than one": {"v0.11.0", "v0.9.0", "v0.10.0", 0, "Updated ws v0.9.0 to v0.10.0: " + installed},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := updateEnv(tc.running, serveRelease(t, tc.latest, "new ws"), 0)
			m.Results[installed+" version"] = effecttest.Result{Out: []byte(tc.installed + "\n")}
			if tc.installed == "" {
				delete(m.Files, installed)
				m.Results[installed+" version"] = effecttest.Result{Err: errors.New("no such file or directory")}
			}
			m.Results[copied+" version"] = effecttest.Result{Out: []byte(tc.latest + "\n")}
			status, stdout, stderr := runUpdate(p, "server", true)
			if status != tc.status || !strings.Contains(stdout+stderr, tc.report) {
				t.Fatalf("status = %d, stdout = %q, stderr = %q; want %d and %q", status, stdout, stderr, tc.status, tc.report)
			}
			if tc.status != 0 {
				if string(m.Files[installed]) != "old ws" || indexOf(m.Calls, "exec ") >= 0 {
					t.Errorf("file %q, calls %v; want the old ws and no setup", m.Files[installed], m.Calls)
				}
				return
			}
			if string(m.Files[installed]) != "new ws" || m.Calls[len(m.Calls)-1] != "exec "+installed+" server setup" {
				t.Errorf("file %q, calls %v; want the release, then setup of the installed ws", m.Files[installed], m.Calls)
			}
		})
	}
}

func indexOf(calls []string, call string) int {
	return slices.IndexFunc(calls, func(c string) bool { return strings.HasPrefix(c, call) })
}

func TestUpdateRefusesBeforeItRunsSetup(t *testing.T) {
	for name, tc := range map[string]struct {
		version, latest string
		args            []string
		euid            int
		fails           string
		status          int
	}{
		"a release after the latest": {"v0.11.0", "v0.10.0", nil, 0, "--version", 1},
		"a user on the server":       {"v0.9.0", "v0.10.0", nil, 501, "run it as root", 1},
		"a version of no release":    {"v0.9.0", "v0.10.0", []string{"--version", "0.10"}, 0, "no release", 2},
		"a pre-release but rc":       {"v0.9.0", "v0.10.0", []string{"--version", "0.10.0-beta.1"}, 0, "no release", 2},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := updateEnv(tc.version, serveRelease(t, tc.latest, "new ws"), tc.euid)
			status, _, stderr := runUpdate(p, "server", true, tc.args...)
			if status != tc.status || !strings.Contains(stderr, tc.fails) {
				t.Errorf("status = %d, stderr = %q; want %d and %q", status, stderr, tc.status, tc.fails)
			}
			if string(m.Files[installed]) != "old ws" || len(m.Calls) > 1 || indexOf(m.Calls, "run "+installed+" version") != len(m.Calls)-1 {
				t.Errorf("file %q, calls %v; want the old ws, and no call but its version", m.Files[installed], m.Calls)
			}
		})
	}
}

func TestUpdateOfTheMacRefusesRoot(t *testing.T) {
	p, m := updateEnv("v0.9.0", serveRelease(t, "v0.10.0", "new ws"), 0)
	if status, _, stderr := runUpdate(p, "client", false); status != 1 || !strings.Contains(stderr, "not root") || len(m.Calls) != 0 {
		t.Errorf("status = %d, stderr = %q, calls %v; want the refusal and no call", status, stderr, m.Calls)
	}
}

func TestUpdateOfAWrongSumRunsNoSetup(t *testing.T) {
	p, m := updateEnv("v0.9.0", serveRelease(t, "v0.10.0", "another ws", "new ws"), 0)
	status, _, stderr := runUpdate(p, "server", true, "--version", "0.10.0")
	if status != 1 || !strings.Contains(stderr, "does not match") {
		t.Errorf("status = %d, stderr = %q; want the checksum's refusal", status, stderr)
	}
	if string(m.Files[installed]) != "old ws" || strings.Join(m.Calls, "\n") != "run "+installed+" version" {
		t.Errorf("file %q, calls %v; want the old ws, and no call but its version", m.Files[installed], m.Calls)
	}
}
