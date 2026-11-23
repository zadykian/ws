package local_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// loggedSudo is a sudo that runs its command bare, and keeps a line of each in the log it returns.
// It fails a chown or chmod whose -- follows an operand, as the Mac's tools read that -- as a file.
func loggedSudo(t *testing.T) (local.OSMachine, string) {
	log := filepath.Join(t.TempDir(), "sudo.log")
	return local.OSMachine{Sudo: []string{"sh", "-c", `printf '%s\n' "$1" >>"$0" && case $1$3 in chown-- | chmod--) exit 64 ;; esac && exec "$@"`, log}}, log
}

func sudoRan(t *testing.T, log string) string {
	data, err := os.ReadFile(log) //nolint:gosec // a temporary path of the test
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(string(data)), " ")
}

func TestAnotherOwnersFilesGoThroughSudo(t *testing.T) {
	m, log := loggedSudo(t)
	local.RunAs(t, os.Getuid()+1, false)
	t.Setenv("HOME", t.TempDir())
	dir, own := t.TempDir(), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	if err := m.WriteAtomic(dir+"/f", []byte("new"), own); err != nil {
		t.Fatal(err)
	}
	if err := m.Mkdir(dir+"/d", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := m.Chown(dir+"/d", own.UID, own.GID); err != nil {
		t.Fatal(err)
	}
	if err := m.Chmod(dir+"/d", 0o700); err != nil {
		t.Fatal(err)
	}
	if got, want := sudoRan(t, log), "sh mkdir chown chmod"; got != want {
		t.Errorf("sudo ran %q, want %q", got, want)
	}
	if got, err := os.ReadFile(dir + "/f"); err != nil || string(got) != "new" { //nolint:gosec // a temporary path of the test
		t.Errorf("content = %q, %v; want new", got, err)
	}
	if info, err := os.Stat(dir + "/d"); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, %v; want 0700", info.Mode(), err)
	}
	if err := m.Mkdir(dir+"/d", 0o750); !os.IsExist(err) {
		t.Errorf("Mkdir of a directory that exists: err = %v, want fs.ErrExist", err)
	}
}

func TestOwnFilesStayOffSudo(t *testing.T) {
	m, log := loggedSudo(t)
	dir, own := t.TempDir(), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	local.RunAs(t, own.UID, true)
	if err := m.WriteAtomic(dir+"/g", []byte("new"), own); err != nil {
		t.Fatal(err)
	}
	if err := m.Mkdir(dir+"/d", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := m.Chmod(dir+"/d", 0o700); err != nil {
		t.Fatal(err)
	}
	if got := sudoRan(t, log); got != "" {
		t.Errorf("sudo ran %q, want nothing", got)
	}
}

func TestHomeAndLinksStayOffSudo(t *testing.T) {
	m, log := loggedSudo(t)
	dir, own := t.TempDir(), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	local.RunAs(t, own.UID+1, false)
	t.Setenv("HOME", dir)
	if err := m.WriteAtomic(dir+"/f", []byte("new"), own); err != nil {
		t.Fatal(err)
	}
	if err := m.Mkdir(dir+"/d", 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir+"/f", link); err != nil {
		t.Fatal(err)
	}
	if err := m.Chown(link, own.UID, own.GID); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", "")
	if err := m.Chmod(dir+"/d", 0o700); err != nil {
		t.Fatal(err)
	}
	if got := sudoRan(t, log); got != "" {
		t.Errorf("sudo ran %q, want nothing", got)
	}
}

func TestADownloadToAnotherOwnersPathGoesThroughSudoOnceTheSumHolds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("body")) }))
	defer srv.Close()
	m, log := loggedSudo(t)
	local.RunAs(t, os.Getuid()+1, false)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	dir, own := t.TempDir(), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o755}
	if err := m.Download(t.Context(), srv.URL, dir+"/f", strings.Repeat("0", 64), own); !errors.Is(err, effect.ErrSHA256) {
		t.Errorf("a wrong sum: err = %v, want an ErrSHA256 error", err)
	}
	if got := sudoRan(t, log); got != "" {
		t.Errorf("a wrong sum: sudo ran %q, want nothing", got)
	}
	if err := m.Download(t.Context(), srv.URL, dir+"/f", fmt.Sprintf("%x", sha256.Sum256([]byte("body"))), own); err != nil {
		t.Fatal(err)
	}
	if got := sudoRan(t, log); got != "sh" {
		t.Errorf("sudo ran %q, want sh", got)
	}
	if got, err := os.ReadFile(dir + "/f"); err != nil || string(got) != "body" { //nolint:gosec // a temporary path of the test
		t.Errorf("content = %q, %v; want body", got, err)
	}
	if info, err := os.Stat(dir + "/f"); err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, %v; want 0755", info, err)
	}
	if left, _ := os.ReadDir(os.TempDir()); len(left) != 0 {
		t.Errorf("the download left %d files in the temporary directory", len(left))
	}
}

func TestALinkInAnotherOwnersDirectoryGoesThroughSudo(t *testing.T) {
	m, log := loggedSudo(t)
	local.RunAs(t, os.Getuid()+1, false)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	for _, target := range []string{dir + "/a", dir + "/b"} {
		if err := m.Symlink(target, dir+"/link"); err != nil {
			t.Fatal(err)
		}
		if got, err := os.Readlink(dir + "/link"); err != nil || got != target {
			t.Errorf("link = %q, %v; want %s", got, err, target)
		}
	}
	if got := sudoRan(t, log); got != "sh sh" {
		t.Errorf("sudo ran %q, want sh sh: one call for each link", got)
	}
}

func TestALinkOverADirectoryInAnotherOwnersDirectoryIsRefused(t *testing.T) {
	m, log := loggedSudo(t)
	local.RunAs(t, os.Getuid()+1, false)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.Mkdir(dir+"/link", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := m.Symlink(dir+"/a", dir+"/link"); err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("a link over a directory: err = %v, want it refused", err)
	}
	if got := sudoRan(t, log); got != "sh" {
		t.Errorf("sudo ran %q, want one sh, which checks as root", got)
	}
	if left, _ := os.ReadDir(dir + "/link"); len(left) != 0 {
		t.Errorf("the directory holds %d entries", len(left))
	}
}
