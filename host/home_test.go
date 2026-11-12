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
