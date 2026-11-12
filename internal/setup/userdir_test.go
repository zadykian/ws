package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// TestUserDirPlanAgreesWithReport finds that a report stands off where a plan changes the
// directory, and has a conflict where the directory is there.
func TestUserDirPlanAgreesWithReport(t *testing.T) {
	const dir = "/h/d"
	for _, c := range []struct {
		setUp func(*effecttest.Machine)
		fix   bool
		// want is the plan: make, mode or nothing.
		name, want string
	}{
		{func(*effecttest.Machine) {}, false, "missing", "make"},
		{func(m *effecttest.Machine) { m.Dirs[dir] = 0o700 }, true, "the same", "nothing"},
		{func(m *effecttest.Machine) { m.Dirs[dir] = 0o755 }, true, "another mode", "mode"},
		{func(m *effecttest.Machine) { m.Dirs[dir] = 0o755 }, false, "another mode left", "nothing"},
		{func(m *effecttest.Machine) { m.Dirs["/t"], m.Links[dir] = 0o700, "/t" }, true, "a link", "nothing"},
	} {
		m := effecttest.NewMachine()
		c.setUp(m)
		d := userDir{M: m, Path: dir, Mode: 0o700, Fix: c.fix}
		cs, _, err := userDirs{d}.plan()
		items, reportErr := d.Report(context.Background())
		got := "nothing"
		switch {
		case len(cs) == 1 && cs[0] == SetMode{dir, 0o700}:
			got = "mode"
		case len(cs) == 1:
			got = "make"
		}
		if err != nil || reportErr != nil || got != c.want {
			t.Errorf("%s: plan = %v, %v, report's error %v, want %s", c.name, cs, err, reportErr, c.want)
			continue
		}
		it := items[0]
		if (it.Found != it.Want) != (got != "nothing") || (it.Conflict != "") != (got == "mode") {
			t.Errorf("%s: report = %+v, plan %s", c.name, it, got)
		}
	}
}

// TestUserDirStopsAtAFileAndReportsItAsAConflict finds the policy: a plan fails where the path is
// no directory, and a report shows why.
func TestUserDirStopsAtAFileAndReportsItAsAConflict(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/d"] = []byte("x")
	d := userDir{M: m, Path: "/h/d", Mode: 0o755}
	if cs, _, err := (userDirs{d}).plan(); !errors.Is(err, errNotDir) || len(cs) != 0 {
		t.Errorf("plan = %v, %v, want errNotDir and no change", cs, err)
	}
	items, err := d.Report(context.Background())
	if err != nil || len(items) != 1 || items[0].Conflict == "" || items[0].Found == items[0].Want {
		t.Errorf("report = %+v, %v, want one conflict", items, err)
	}
}

func TestUserDirsGiveTheLastLinkAndStopAtTheFirstError(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/t"], m.Dirs["/h"], m.Links["/a"], m.Links["/h/b"] = 0o755, 0o755, "/t", "/t"
	ds := userDirs{{M: m, Path: "/a", Mode: 0o755}, {M: m, Path: "/h", Mode: 0o755}, {M: m, Path: "/h/b", Mode: 0o755}}
	cs, link, err := ds.plan()
	if err != nil || len(cs) != 0 || link != (dirLink{"/h/b", "/t"}) {
		t.Errorf("plan = %v, %v, %v, want the last link and no change", cs, link, err)
	}
	m.Files["/h/b"] = []byte("x")
	delete(m.Links, "/h/b")
	if _, _, err := ds.plan(); !errors.Is(err, errNotDir) {
		t.Errorf("plan's error = %v, want errNotDir", err)
	}
}
