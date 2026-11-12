package update

import (
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
	file   = "/h/.local/bin/ws"
	copied = "/h/.local/bin/.ws.7"
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
	t.Helper()
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

// newUpdater returns an Updater of a ws at version in file, whose download prints printed.
func newUpdater(version, url, printed string) (Updater, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files[file] = oldBinary
	m.Modes[file] = 0o750
	m.Results[copied+" version"] = effecttest.Result{Out: []byte(printed + "\n")}
	m.Results["rm -f "+copied] = effecttest.Result{}
	return Updater{M: m, Version: version, URL: url, File: file, Binary: binary, PID: 7}, m
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

func TestReplace(t *testing.T) {
	u, m := newUpdater("v1.0.0", release(t, "v2.0.0", good()), "v2.0.0")
	if err := u.replace(t.Context(), "v2.0.0"); err != nil {
		t.Fatalf("replace = %v", err)
	}
	if got := string(m.Files[file]); got != string(newBinary) {
		t.Errorf("the file holds %q, want the new binary", got)
	}
	if got := m.Modes[file]; got != 0o750 {
		t.Errorf("the file's mode is %o, want the old one's, 750", got)
	}
	if got := m.Owners[file]; got != [2]int{-1, -1} {
		t.Errorf("the file's owner is %v, want the user who runs ws", got)
	}
	want := []string{"stat " + file, "atomic " + copied + " 750", "run " + copied + " version",
		"atomic " + file + " 750", "run rm -f " + copied}
	if got := strings.Join(m.Calls, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("replace made the calls\n%s\nwant\n%s", got, strings.Join(want, "\n"))
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
	// v0.10.0 is the later by number, and the earlier as text.
	u, m := newUpdater("v0.9.0", release(t, "v0.10.0", good()), "v0.10.0")
	r, err := u.Run(t.Context())
	if err != nil || r.Report() != "Updated ws v0.9.0 to v0.10.0: "+file+"\n" {
		t.Fatalf("Run = %+v, %v", r, err)
	}
	if got := string(m.Files[file]); got != string(newBinary) {
		t.Errorf("the file holds %q, want the new binary", got)
	}
}

func TestRunStays(t *testing.T) {
	for name, tc := range map[string]struct{ version, latest, report string }{
		"the latest": {"v1.2.3", "v1.2.3", "ws v1.2.3 is the latest release\n"},
		"newer":      {"v0.10.0", "v0.9.0", "ws v0.10.0 is newer than the latest release, v0.9.0\n"},
	} {
		t.Run(name, func(t *testing.T) {
			// The release holds no file: a download would fail.
			u, m := newUpdater(tc.version, release(t, tc.latest, nil), tc.latest)
			r, err := u.Run(t.Context())
			if err != nil || r.Report() != tc.report || len(m.Calls) != 0 {
				t.Errorf("Run = %+v, %v, calls %v; want %q and no call", r, err, m.Calls, tc.report)
			}
		})
	}
}

func TestRunRefuses(t *testing.T) {
	for name, version := range map[string]string{
		"devel":  "devel",
		"pseudo": "v1.2.4-0.20260101000000-abcdef123456",
		"dirty":  "v1.2.3+dirty",
	} {
		t.Run(name, func(t *testing.T) {
			u, m := newUpdater(version, "http://127.0.0.1:1", "v9.9.9")
			if _, err := u.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "not a release") || len(m.Calls) != 0 {
				t.Errorf("Run = %v, calls %v; want the refusal, and no call", err, m.Calls)
			}
		})
	}
	u, _ := newUpdater("v1.2.3", "http://example.com", "v9.9.9")
	if _, err := u.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "WS_RELEASES_URL") {
		t.Errorf("Run at another address = %v, want the refusal", err)
	}
}
