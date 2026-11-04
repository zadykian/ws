package update

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
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
