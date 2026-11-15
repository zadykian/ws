package file

import (
	"bytes"
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func homeLinesIn(m *effecttest.Machine, top bool) HomeLines {
	r := linesIn(m, top)
	r.Path = "/home/ws/f"
	return HomeLines{Rules: r, UID: 1000, GID: 1001, Home: "/home/ws"}
}

func TestHomeLinesReportsHoldingMissingAndMisplaced(t *testing.T) {
	for _, c := range []struct {
		name, text, found, conflict string
		exists                      bool
	}{
		{"holding", "a\nws\n", "holds ws", "", true},
		{"missing", "", "missing", "", false},
		{"lacking", "a\n", "lacks it", "", true},
		{"out of place", "late\nws\n", "out of place", "ws is below \"late\"", true},
	} {
		m := effecttest.NewMachine()
		if c.exists {
			m.Files["/home/ws/f"] = []byte(c.text)
		}
		items, err := homeLinesIn(m, true).Report(context.Background())
		if err != nil || items[0].Found != c.found || items[0].Conflict != c.conflict {
			t.Errorf("%s: %+v, %v", c.name, items, err)
		}
	}
}

func TestHomeLinesReportsALinkAtTheFileEvenWhereItHoldsTheLinesAndReadsNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/real"], m.Links["/home/ws/f"] = []byte("ws\n"), "/h/real"
	l := homeLinesIn(m, false)
	l.Rules.Has = func(context.Context, effect.Machine, string, []byte) (bool, error) {
		t.Error("Has read through a link")
		return true, nil
	}
	items, err := l.Report(context.Background())
	want := "a link to /h/real: add 'ws' at the end of that file"
	if err != nil || items[0].Conflict != want {
		t.Errorf("%+v, %v; want %q", items, err, want)
	}
	if bytes.Contains([]byte(joinCalls(m)), []byte("read root /h/real")) {
		t.Errorf("read the target: %v", m.Calls)
	}
}

func TestHomeLinesRefusesALinkOnTheWay(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links["/home/ws"] = "/srv"
	l := homeLinesIn(m, true)
	l.Rules.Path = "/home/ws/d/f"
	m.Links["/home/ws/d"] = "/srv/d"
	if _, err := l.Report(context.Background()); err == nil {
		t.Error("a link on the way was read")
	}
}

// linkHidden is a machine whose Stat misses the link at a path, as a link swapped in after the check.
type linkHidden struct {
	*effecttest.Machine
	at string
}

func (h linkHidden) Stat(path string) (effect.Info, error) {
	if path == h.at {
		return effect.Info{Mode: 0o644}, nil
	}
	return h.Machine.Stat(path)
}

func TestHomeLinesReadsNoFileThroughALinkSwappedInAfterTheCheck(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/shadow"], m.Links["/home/ws/f"] = []byte("root:x\n"), "/etc/shadow"
	l := homeLinesIn(m, true)
	l.Rules.M = linkHidden{m, "/home/ws/f"}
	if items, err := l.Report(context.Background()); err == nil {
		t.Errorf("Report read through a link swapped in after the check: %+v", items)
	}
}

func joinCalls(m *effecttest.Machine) string {
	var b bytes.Buffer
	for _, c := range m.Calls {
		b.WriteString(c + "\n")
	}
	return b.String()
}
