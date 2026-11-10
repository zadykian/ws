package setup

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

// TestUserFilePlanAgreesWithReport finds that a report stands off where a plan changes the file or
// refuses it, has a conflict where that file is there, and fails where the plan fails.
func TestUserFilePlanAgreesWithReport(t *testing.T) {
	const file = "/h/f"
	put := func(text string, mode fs.FileMode) func(*effecttest.Machine) {
		return func(m *effecttest.Machine) { m.Files[file], m.Modes[file] = []byte(text), mode }
	}
	for _, c := range []struct {
		setUp func(*effecttest.Machine)
		mode  fs.FileMode
		// want is the plan: a write, a mode, nothing, not ws's or fails.
		name, want string
	}{
		{func(*effecttest.Machine) {}, 0o600, "missing", "write"},
		{put("# ws\n", 0o600), 0o600, "the same", "nothing"},
		{put("# ws\n", 0o640), 0o600, "another mode", "mode"},
		{put("# ws\n", 0o640), 0, "a mode left to the user", "nothing"},
		{put("# ws\nmore\n", 0o600), 0o600, "other content", "write"},
		{put("mine\n", 0o600), 0o600, "another's", "not ws's"},
		{func(m *effecttest.Machine) { m.Links[file] = "/nowhere" }, 0o600, "a link to nowhere", "not ws's"},
		{func(m *effecttest.Machine) { m.Dirs[file] = 0o755 }, 0o600, "a directory", "not ws's"},
		{func(m *effecttest.Machine) { put("# ws\n", 0o600)(m); m.Root[file] = effect.Attrs{} }, 0o600, "unreadable", "fails"},
	} {
		m := effecttest.NewMachine()
		c.setUp(m)
		f := userFile{M: m, Path: file, Text: []byte("# ws\n"), Mode: c.mode,
			Own: func(have []byte) bool { return bytes.HasPrefix(have, []byte("# ws")) }}
		cs, planErr := f.plan()
		items, reportErr := f.Report(context.Background())
		got := "nothing"
		switch {
		case errors.Is(planErr, errNotWs):
			got = "not ws's"
		case planErr != nil:
			got = "fails"
		case len(cs) == 1 && cs[0] == SetMode{file, c.mode}:
			got = "mode"
		case len(cs) == 1:
			got = "write"
		}
		if got != c.want || (got == "fails") != (reportErr != nil) {
			t.Errorf("%s: plan = %v, %v, report's error %v, want %s", c.name, cs, planErr, reportErr, c.want)
			continue
		}
		if reportErr == nil {
			it := items[0]
			if (it.Found != it.Want) != (got != "nothing") || (it.Conflict != "") != (got != "nothing" && c.name != "missing") {
				t.Errorf("%s: report = %+v, plan %s", c.name, it, got)
			}
		}
	}
}
