package setup

import (
	"context"
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
