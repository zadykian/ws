package local_test

import (
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

func TestOwnAndConfinedFilesStayOffSudo(t *testing.T) {
	m, log := loggedSudo(t)
	dir, own := t.TempDir(), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	local.RunAs(t, own.UID+1, false)
	m.Confine = dir
	if err := m.WriteAtomic(dir+"/f", []byte("new"), own); err != nil {
		t.Fatal(err)
	}
	m.Confine = ""
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
