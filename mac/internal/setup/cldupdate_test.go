package setup_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestCldCheckNotesAnOlderRelease(t *testing.T) {
	s, m, _ := cldStep(true)
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.14.0")}
	got := strings.Join(cldChanges(t, s), "\n")
	want := "would run cld update: cld 0.13.0, the latest 0.14.0\n" + cldLine
	if got != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestCldRunUpdatesAndAGhosttyAndTabsMakeNoNote(t *testing.T) {
	s, m, _ := cldStep(false)
	m.Dirs["/Applications/Ghostty.app"] = 0o755
	if got := cldChanges(t, s); len(got) != 1 || got[0] != "would run cld update" {
		t.Errorf("Plan = %q; want the update alone", got)
	}
}

func TestCldWarnsWhereItsVersionFailsOrIsNoRelease(t *testing.T) {
	for out, warns := range map[string]int{"": 1, "cld dev\n": 0} {
		s, m, o := cldStep(false)
		m.Results[cldBin+" --version"] = effecttest.Result{Out: []byte(out)}
		if out == "" {
			m.Results[cldBin+" --version"] = effecttest.Result{Err: errors.New(cldNo)}
		}
		if got := cldChanges(t, s); len(got) != 0 || len(o.Lines) != warns {
			t.Errorf("version %q: Plan = %q, out %q; want none and %d warnings", out, got, o.Lines, warns)
		}
	}
}

func TestCldWithoutUpdateIsReplacedByTheInstall(t *testing.T) {
	s, m, _ := cldStep(false)
	m.Results[cldBin+" help update"] = effecttest.Result{Err: errors.New(cldNo)}
	if got := cldChanges(t, s); len(got) != 1 || !strings.HasPrefix(got[0], "would install cld") {
		t.Errorf("Plan = %q; want the install", got)
	}
}

func TestCldCheckWithTheLatestAndNoTabsPlansNothing(t *testing.T) {
	s, m, _ := cldStep(true)
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.13.0")}
	if got := cldChanges(t, s); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}
