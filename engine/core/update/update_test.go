package update

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	file   = "/usr/local/bin/ws"
	copied = "/usr/local/bin/.ws.7"
	binary = "ws-linux-amd64"
)

var (
	oldBinary = []byte("old ws")
	newBinary = []byte("new ws")
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// release serves a latest release tag, with files by their names.
func release(t *testing.T, tag string, files map[string][]byte) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
			return
		}
		name, ok := strings.CutPrefix(r.URL.Path, "/download/"+tag+"/")
		body, found := files[name]
		if !ok || !found {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// good is the files of a release with a binary that matches its checksum.
func good() map[string][]byte {
	return map[string][]byte{binary: newBinary, "ws.sha256": []byte(sum(newBinary) + "  " + binary + "\n")}
}

// newUpdater returns an Updater of a file that prints version, whose download prints printed.
func newUpdater(version, url, printed string) (Updater, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files[file] = oldBinary
	m.Results[file+" version"] = effecttest.Result{Out: []byte(version + "\n")}
	m.Results[copied+" version"] = effecttest.Result{Out: []byte(printed + "\n")}
	return Updater{M: m, URL: url, File: file, Binary: binary, PID: 7}, m
}

// onlyItsVersion reports whether calls hold no call but the run of the file's version.
func onlyItsVersion(calls []string) bool {
	return len(calls) == 1 && calls[0] == "run "+file+" version"
}

func TestDownload(t *testing.T) {
	u, _ := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "v2.0.0")
	got, err := u.download(t.Context(), u.URL+"/download/v2.0.0/")
	if err != nil || string(got) != string(newBinary) {
		t.Errorf("download = %q, %v; want the binary", got, err)
	}
}

func TestDownloadFails(t *testing.T) {
	wrong := good()
	wrong[binary] = []byte("another binary")
	for name, tc := range map[string]struct {
		files map[string][]byte
		fails string
	}{
		"no ws.sha256":     {map[string][]byte{binary: newBinary}, "404"},
		"no line for it":   {map[string][]byte{"ws.sha256": []byte(sum(newBinary) + "  ws-other\n")}, "no checksum for"},
		"no binary":        {map[string][]byte{"ws.sha256": good()["ws.sha256"]}, "404"},
		"another checksum": {wrong, "does not match"},
	} {
		t.Run(name, func(t *testing.T) {
			u, m := newUpdater("v1.0.0", release(t, "v2.0.0", tc.files), "v2.0.0")
			_, err := u.download(t.Context(), u.URL+"/download/v2.0.0/")
			if err == nil || !strings.Contains(err.Error(), tc.fails) || len(m.Calls) != 0 {
				t.Errorf("download = %v, calls %v; want an error of %q, and no call", err, m.Calls, tc.fails)
			}
		})
	}
}

// checked is what the copy holds once it has run: the file that printed the tag.
var checked = []byte("the copy that printed the tag")

// checking is a Machine whose run of the copy marks it, so that a test tells the copy that printed
// the tag from the download's bytes written again.
type checking struct{ *effecttest.Machine }

func (c checking) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == copied {
		c.Files[copied] = checked
	}
	return c.Machine.Run(ctx, name, args...)
}

func TestReplaceRenamesTheCopyItChecked(t *testing.T) {
	u, m := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "v2.0.0")
	u.M = checking{m}
	if err := u.replace(t.Context(), "v2.0.0"); err != nil {
		t.Fatalf("replace = %v", err)
	}
	if got := string(m.Files[file]); got != string(checked) {
		t.Errorf("the file holds %q, want the copy that printed the tag", got)
	}
	if got := m.Modes[file]; got != 0o755 {
		t.Errorf("the file's mode is %o, want 755", got)
	}
	if got := m.Owners[file]; got != [2]int{0, 0} {
		t.Errorf("the file's owner is %v, want root's", got)
	}
	if _, left := m.Files[copied]; left {
		t.Error("the copy stays")
	}
	want := []string{"atomic " + copied + " 755", "run " + copied + " version", "stat " + file,
		"root mv -f -- " + copied + " " + file}
	if got := strings.Join(m.Calls, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("replace made the calls\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

// TestReplaceKeepsWhatStandsAtTheFile checks that a directory at the file, which mv would move the
// copy into, and a rename that fails leave the file, and no copy.
func TestReplaceKeepsWhatStandsAtTheFile(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(m *effecttest.Machine)
		fails  string
	}{
		"a directory": {func(m *effecttest.Machine) {
			delete(m.Files, file)
			m.Dirs[file] = 0o755
		}, "is a directory"},
		"a failed rename": {func(m *effecttest.Machine) {
			m.Results["mv -f -- "+copied+" "+file] = effecttest.Result{Err: errors.New("read-only file system")}
		}, "read-only file system"},
	} {
		t.Run(name, func(t *testing.T) {
			u, m := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "v2.0.0")
			tc.change(m)
			before := string(m.Files[file])
			err := u.replace(t.Context(), "v2.0.0")
			if err == nil || !strings.Contains(err.Error(), tc.fails) {
				t.Errorf("replace = %v, want an error of %q", err, tc.fails)
			}
			if _, left := m.Files[copied]; left || string(m.Files[file]) != before {
				t.Errorf("file %q, copy left %v; want the file as it was, and no copy", m.Files[file], left)
			}
		})
	}
}

// TestReplaceFails checks that a binary that fails its run leaves ws unchanged, and no copy.
func TestReplaceFails(t *testing.T) {
	for name, tc := range map[string]struct {
		run   effecttest.Result
		fails string
	}{
		"another version": {effecttest.Result{Out: []byte("v1.0.0\n")}, `is version "v1.0.0"`},
		"does not run":    {effecttest.Result{Err: errors.New("exec format error")}, "does not run"},
	} {
		t.Run(name, func(t *testing.T) {
			u, m := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "")
			m.Results[copied+" version"] = tc.run
			err := u.replace(t.Context(), "v2.0.0")
			calls := strings.Join(m.Calls, "\n")
			if err == nil || !strings.Contains(err.Error(), tc.fails) {
				t.Errorf("replace = %v, want an error of %q", err, tc.fails)
			}
			if string(m.Files[file]) != string(oldBinary) || !strings.Contains(calls, "rm -f "+copied) {
				t.Errorf("file %q, calls %q; want the old file and the copy removed", m.Files[file], calls)
			}
		})
	}
}

// cancelling is a Machine whose run of the copy cancels the update, as a signal does.
type cancelling struct {
	*effecttest.Machine
	cancel context.CancelFunc
}

func (c cancelling) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == copied {
		c.cancel()
	}
	return c.Machine.Run(ctx, name, args...)
}

func TestReplaceCancelled(t *testing.T) {
	u, m := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "v2.0.0")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	u.M = cancelling{m, cancel}
	err := u.replace(ctx, "v2.0.0")
	calls := strings.Join(m.Calls, "\n")
	if !errors.Is(err, context.Canceled) || string(m.Files[file]) != string(oldBinary) || !strings.Contains(calls, "rm -f "+copied) {
		t.Errorf("replace = %v, file %q, calls %q; want the old file and the copy removed", err, m.Files[file], calls)
	}
}

func TestRunReplaces(t *testing.T) {
	for name, tc := range map[string]struct{ version, tag, latest string }{
		// v0.10.0 is the later by number, and the earlier as text.
		"the latest":           {"v0.9.0", "", "v0.10.0"},
		"an rc to the release": {"v0.10.0-rc.2", "", "v0.10.0"},
		"a trial's build":      {"v0.9.1-0.20261101000000-abcdef123456", "", "v0.10.0"},
		"a build of no tag":    {"devel", "", "v0.10.0"},
		"a named release":      {"v0.9.0", "v0.10.0", "v0.9.5"},
		"a named rc":           {"v0.9.0", "v0.10.0-rc.1", "v0.9.5"},
		"a named older":        {"v0.10.0", "v0.9.0", "v0.10.0"},
	} {
		t.Run(name, func(t *testing.T) {
			to := cmp.Or(tc.tag, tc.latest)
			u, m := newUpdater(tc.version, release(t, to, good()), to)
			u.Tag = tc.tag
			r, err := u.Run(t.Context())
			if err != nil || r.Report() != "Updated ws "+tc.version+" to "+to+": "+file+"\n" {
				t.Fatalf("Run = %+v, %v", r, err)
			}
			if got := string(m.Files[file]); got != string(newBinary) {
				t.Errorf("the file holds %q, want the new binary", got)
			}
		})
	}
}

func TestRunStays(t *testing.T) {
	for name, tc := range map[string]struct{ version, tag, latest string }{
		"the latest":  {"v1.2.3", "", "v1.2.3"},
		"a named one": {"v1.2.3-rc.1", "v1.2.3-rc.1", "v1.2.0"},
	} {
		t.Run(name, func(t *testing.T) {
			// The release holds no file: a download would fail.
			u, m := newUpdater(tc.version, release(t, tc.latest, nil), tc.latest)
			u.Tag = tc.tag
			r, err := u.Run(t.Context())
			if err != nil || r.Report() != file+" is ws "+tc.version+" already\n" || !onlyItsVersion(m.Calls) {
				t.Errorf("Run = %+v, %v, calls %v; want the release already, and no call but its version", r, err, m.Calls)
			}
		})
	}
}

func TestRunRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		version, tag, latest, fails string
		calls                       int
	}{
		"newer than the latest": {"v0.10.0", "", "v0.9.0", file + " is v0.10.0, later", 1},
		"an rc after it":        {"v0.10.0-rc.1", "", "v0.9.0", "--version", 1},
		"no tag":                {"v1.2.3", "1.2.4", "v1.2.4", "no release's tag", 0},
		"a beta":                {"v1.2.3", "v1.2.4-beta.1", "v1.2.4", "no release's tag", 0},
	} {
		t.Run(name, func(t *testing.T) {
			u, m := newUpdater(tc.version, release(t, tc.latest, good()), tc.latest)
			u.Tag = tc.tag
			_, err := u.Run(t.Context())
			if err == nil || !strings.Contains(err.Error(), tc.fails) || len(m.Calls) != tc.calls || tc.calls > 0 && !onlyItsVersion(m.Calls) {
				t.Errorf("Run = %v, calls %v; want a refusal naming %q, and no call but its version", err, m.Calls, tc.fails)
			}
			if string(m.Files[file]) != string(oldBinary) {
				t.Errorf("the file holds %q, want the old one", m.Files[file])
			}
		})
	}
	u, _ := newUpdater("v1.2.3", "http://example.com", "v9.9.9")
	if _, err := u.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "WS_RELEASES_URL") {
		t.Errorf("Run at another address = %v, want the refusal", err)
	}
}
