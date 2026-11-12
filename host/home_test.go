package host_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

var ws = host.User{Name: "ws", Home: "/home/ws"}

func TestReadHomeReadsAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/.config/x/f"] = []byte("body")
	got, err := ws.ReadHome(m, ".config/x/f")
	if err != nil || string(got) != "body" {
		t.Errorf("ReadHome = %q, %v", got, err)
	}
}

func TestReadHomeGivesNotExistForAMissingFile(t *testing.T) {
	if _, err := ws.ReadHome(effecttest.NewMachine(), ".config/x/f"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadHome = %v, want fs.ErrNotExist", err)
	}
}

func TestReadHomeRefusesALinkAtTheFileAndAtEachDirectory(t *testing.T) {
	for _, link := range []string{"/home/ws/.config", "/home/ws/.config/x", "/home/ws/.config/x/f"} {
		m := effecttest.NewMachine()
		m.Files["/srv/target/x/f"] = []byte("body")
		m.Links[link] = "/srv/target"
		_, err := ws.ReadHome(m, ".config/x/f")
		want := link + " is a link to /srv/target: replace it with the file itself"
		if err == nil || err.Error() != want {
			t.Errorf("link at %s: %v, want %q", link, err, want)
		}
	}
}

func TestReadHomeFailsWhereItCannotLookForALink(t *testing.T) {
	m := statFails{effecttest.NewMachine(), "/home/ws/.config"}
	m.Files["/home/ws/.config/x/f"] = []byte("body")
	if _, err := ws.ReadHome(m, ".config/x/f"); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("ReadHome = %v; want Stat's error", err)
	}
}

func TestReadHomeRefusesAPathThatIsNotLocalBeforeAnyCall(t *testing.T) {
	for _, rel := range []string{"../x", "a/../../x", "/etc/passwd", ""} {
		m := effecttest.NewMachine()
		if _, err := ws.ReadHome(m, rel); err == nil || !strings.Contains(err.Error(), "not a path below the home") {
			t.Errorf("ReadHome(%q) = %v", rel, err)
		}
		if len(m.Calls) != 0 {
			t.Errorf("ReadHome(%q) called %v", rel, m.Calls)
		}
	}
}

var account = host.Account{User: ws, UID: 1000, GID: 1001}

func TestHomeFileRefusesALinkAsItPlansAndReports(t *testing.T) {
	for _, link := range []string{"/home/ws/.config", "/home/ws/.config/x", "/home/ws/.config/x/f"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		f := account.File(m, ".config/x/f", []byte("a"), 0o644)
		if _, err := f.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), link+" is a link to") {
			t.Errorf("link at %s: Plan = %v", link, err)
		}
		if _, err := f.Report(t.Context()); err == nil {
			t.Errorf("link at %s: Report found no error", link)
		}
	}
}

func TestHomeFileRefusesARelThatIsNotLocal(t *testing.T) {
	for _, rel := range []string{"../x", "/etc/x", ""} {
		f := account.File(effecttest.NewMachine(), rel, nil, 0o644)
		if _, err := f.Plan(t.Context()); err == nil {
			t.Errorf("rel %q: Plan found no error", rel)
		}
	}
}

func TestHomeFilePlansAsTheAccount(t *testing.T) {
	m := effecttest.NewMachine()
	f := account.File(m, ".config/x/f", []byte("a"), 0o644)
	got, err := f.Plan(t.Context())
	if err != nil || len(got) != 1 || !strings.HasPrefix(got[0].String(), "create /home/ws/.config/x/f") {
		t.Fatalf("Plan = %v, %v", got, err)
	}
	m.Files[f.Path] = []byte("a")
	m.Modes[f.Path] = 0o644
	m.Owners[f.Path] = [2]int{1000, 1001}
	if got, err := f.Plan(t.Context()); err != nil || got != nil {
		t.Errorf("second Plan = %v, %v; want none", got, err)
	}
	m.Owners[f.Path] = [2]int{0, 0}
	if got, _ := f.Plan(t.Context()); len(got) != 1 || !strings.Contains(got[0].String(), "set owner 1000:1001") {
		t.Errorf("a file owned by root: Plan = %v", got)
	}
}
