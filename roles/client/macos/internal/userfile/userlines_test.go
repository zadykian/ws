package userfile

import (
	"bytes"
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/resource/file"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func linesIn(m *effecttest.Machine, top bool) Lines {
	return Lines{
		M: m, Path: "/h/f", Lines: "ws\n", Shown: "ws", Top: top, Mode: 0o600,
		Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			if bytes.HasPrefix(text, []byte("late")) {
				return false, Misplaced("ws is below %q", "late")
			}
			return bytes.Contains(text, []byte("ws\n")), nil
		},
	}
}

func TestUserLinesPlanAgreesWithReport(t *testing.T) {
	put := func(text string) func(*effecttest.Machine) {
		return func(m *effecttest.Machine) { m.Files["/h/f"], m.Modes["/h/f"] = []byte(text), 0o640 }
	}
	link := func(text string) func(*effecttest.Machine) {
		return func(m *effecttest.Machine) { put(text)(m); m.Links["/h/f"] = "/h/real" }
	}
	for _, c := range []struct {
		name, want string
		setUp      func(*effecttest.Machine)
		top        bool
	}{
		{"missing", "write", func(*effecttest.Machine) {}, true},
		{"holds", "nothing", put("a\nws\n"), true},
		{"lacks at the top", "write", put("a\n"), true},
		{"lacks at the end", "write", put("a"), false},
		{"a link that lacks", "linked", link("a\n"), false},
		{"a link that holds", "nothing", link("ws\n"), false},
		{"out of place", "misplaced", put("late\nws\n"), true},
	} {
		m := effecttest.NewMachine()
		c.setUp(m)
		l := linesIn(m, c.top)
		cs, planErr := l.Plan(context.Background())
		items, err := l.Report(context.Background())
		if err != nil {
			t.Fatalf("%s: report: %v", c.name, err)
		}
		got := "nothing"
		switch {
		case errors.Is(planErr, ErrLinked):
			got = "linked"
		case errors.Is(planErr, ErrMisplaced):
			got = "misplaced"
		case len(cs) > 0:
			got = "write"
		}
		it := items[0]
		if got != c.want || (it.Conflict != "") != (planErr != nil) || (it.Found != it.Want) != (got != "nothing") {
			t.Errorf("%s: plan %v, %v, report %+v; want %s", c.name, cs, planErr, it, c.want)
		}
	}
}

func TestUserLinesPutTheLinesAtTheTopOrTheEnd(t *testing.T) {
	for _, c := range []struct {
		top        bool
		have, want string
	}{
		{true, "a\n", "ws\n\na\n"},
		{false, "a\n", "a\nws\n"},
		{false, "a", "a\nws\n"},
		{false, "", "ws\n"},
	} {
		m := effecttest.NewMachine()
		m.Files["/h/f"], m.Modes["/h/f"] = []byte(c.have), 0o640
		cs, err := linesIn(m, c.top).Plan(context.Background())
		w, ok := cs[len(cs)-1].(file.UserWrite)
		if err != nil || !ok || string(w.After) != c.want || w.Mode != 0o640 {
			t.Errorf("top %v on %q: %v, %v; want %q", c.top, c.have, cs, err, c.want)
		}
	}
}

func TestUserLinesKeepTheOldFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/f"] = []byte("a\n")
	cs, err := linesIn(m, true).Plan(context.Background())
	want := file.UserBackup{Path: "/h/f", To: "/h/f.20260102030405~"}
	if err != nil || len(cs) != 2 || cs[0] != step.Change(want) {
		t.Errorf("changes = %v, %v; want %v first", cs, err, want)
	}
}
