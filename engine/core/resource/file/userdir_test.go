package file

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// TestUserDirPlanAgreesWithReport finds that a report stands off where a plan changes the
// directory, and shows no change where the directory is there, whatever its mode.
func TestUserDirPlanAgreesWithReport(t *testing.T) {
	const dir = "/h/d"
	for _, c := range []struct {
		setUp func(*effecttest.Machine)
		// want is the plan: make or nothing.
		name, want string
	}{
		{func(*effecttest.Machine) {}, "missing", "make"},
		{func(m *effecttest.Machine) { m.Dirs[dir] = 0o700 }, "the same", "nothing"},
		{func(m *effecttest.Machine) { m.Dirs[dir] = 0o755 }, "another mode left", "nothing"},
		{func(m *effecttest.Machine) { m.Dirs["/t"], m.Links[dir] = 0o700, "/t" }, "a link", "nothing"},
	} {
		m := effecttest.NewMachine()
		c.setUp(m)
		d := UserDir{M: m, Path: dir, Mode: 0o700}
		cs, err := d.Plan(context.Background())
		items, reportErr := d.Report(context.Background())
		got := "nothing"
		if len(cs) == 1 {
			got = "make"
		}
		if err != nil || reportErr != nil || got != c.want {
			t.Errorf("%s: plan = %v, %v, report's error %v, want %s", c.name, cs, err, reportErr, c.want)
			continue
		}
		it := items[0]
		if (it.Found != it.Want) != (got != "nothing") || it.Conflict != "" {
			t.Errorf("%s: report = %+v, plan %s", c.name, it, got)
		}
	}
}

// TestUserDirStopsAtAFileAndReportsItAsAConflict finds the policy: a plan fails where the path is
// no directory, and a report shows why.
func TestUserDirStopsAtAFileAndReportsItAsAConflict(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/d"] = []byte("x")
	d := UserDir{M: m, Path: "/h/d", Mode: 0o755}
	if cs, err := d.Plan(context.Background()); !errors.Is(err, errNotDir) || len(cs) != 0 {
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
	ds := UserDirs{{M: m, Path: "/a", Mode: 0o755}, {M: m, Path: "/h", Mode: 0o755}, {M: m, Path: "/h/b", Mode: 0o755}}
	cs, link, err := ds.Plan()
	if err != nil || len(cs) != 0 || link != (UserDirLink{"/h/b", "/t"}) {
		t.Errorf("plan = %v, %v, %v, want the last link and no change", cs, link, err)
	}
	m.Files["/h/b"] = []byte("x")
	delete(m.Links, "/h/b")
	if _, _, err := ds.Plan(); !errors.Is(err, errNotDir) {
		t.Errorf("plan's error = %v, want errNotDir", err)
	}
}
