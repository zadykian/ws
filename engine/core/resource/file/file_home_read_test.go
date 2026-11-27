package file

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// homeFileAt is a File below /home/ws on a machine whose reads by name answer a link's target.
func homeFileAt(m *effecttest.Machine) File {
	m.Files["/home/ws/f"], m.Modes["/home/ws/f"] = []byte("mine\n"), 0o640
	m.Owners["/home/ws/f"] = [2]int{1002, 1003}
	return File{M: rootReads{m}, Path: "/home/ws/f", Content: []byte("new\n"), Mode: 0o600, UID: 1002, GID: 1003, Home: "/home/ws"}
}

func TestHomeFilePlanReadsThroughTheOneOpen(t *testing.T) {
	m := effecttest.NewMachine()
	cs, err := homeFileAt(m).Plan(context.Background())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want one write", cs, err)
	}
	if got := cs[0].String(); strings.Contains(got, "root:x") || !strings.Contains(got, "mine") {
		t.Errorf("the plan shows %q; want the file's own text", got)
	}
	if calls := joinCalls(m); !strings.Contains(calls, "read home /home/ws/f") || strings.Contains(calls, "read root") {
		t.Errorf("calls %v; want one read of the home file and none by name", m.Calls)
	}
}

func TestHomeFileBackupCopiesTheBytesAndAttributesOfOneRead(t *testing.T) {
	defer FixBackupName("1")()
	m := effecttest.NewMachine()
	f := homeFileAt(m)
	f.Backup = true
	cs, err := f.Plan(context.Background())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v", cs, err)
	}
	if err := cs[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	kept := "/home/ws/f.1~"
	if string(m.Files[kept]) != "mine\n" || m.Owners[kept] != [2]int{1002, 1003} || m.Modes[kept] != 0o640 {
		t.Errorf("the copy holds %q, owner %v, mode %o; want the file's own", m.Files[kept], m.Owners[kept], m.Modes[kept])
	}
}

// linkAfterGuard hides a link from Stat, as a link swapped in after the guard is hidden from it:
// the guard passes, and the read alone meets the link.
type linkAfterGuard struct{ rootReads }

func (l linkAfterGuard) Stat(path string) (effect.Info, error) {
	info, err := l.rootReads.Stat(path)
	info.Link = ""
	return info, err
}

func TestHomeFileBackupCopiesNoLinkSwappedInAfterTheGuard(t *testing.T) {
	defer FixBackupName("1")()
	m := effecttest.NewMachine()
	f := homeFileAt(m)
	f.Backup = true
	f.M = linkAfterGuard{rootReads{m}}
	m.Links["/home/ws/f"] = "/etc/shadow"
	err := (writeFile{f: f}).Apply(context.Background())
	if err == nil || strings.Contains(err.Error(), "is a link to") {
		t.Errorf("Apply = %v; want the read's refusal, past the guard", err)
	}
	if calls := joinCalls(m); !strings.Contains(calls, "read home /home/ws/f") {
		t.Errorf("calls %v; want the backup's read of the home file", m.Calls)
	}
	for path, text := range m.Files {
		if strings.Contains(string(text), "root:x") {
			t.Errorf("%s holds the target of the link", path)
		}
	}
}

func TestRootFileReadsAsRootAsBefore(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/srv/a"] = []byte("old\n")
	f := File{M: m, Path: "/srv/a", Content: []byte("new\n"), Mode: 0o600}
	if _, err := f.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls := joinCalls(m); !strings.Contains(calls, "read root /srv/a") || strings.Contains(calls, "read home") {
		t.Errorf("calls %v; want a read by name", m.Calls)
	}
	f.Path = "/srv/missing"
	if _, _, err := f.read(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("read of a missing file = %v, want fs.ErrNotExist", err)
	}
}
