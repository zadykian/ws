package file

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func homeLinesIn(m *effecttest.Machine, top bool) HomeLines {
	return HomeLines{
		M: m, Path: "/home/ws/f", Lines: "ws\n", Shown: "ws", Top: top, Mode: 0o600,
		Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			return bytes.Contains(text, []byte("ws\n")), nil
		},
		UID: 1000, GID: 1001, Home: "/home/ws",
	}
}

func TestHomeLinesFailsOnAHasError(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("late\nws\n")
	l := homeLinesIn(m, true)
	boom := errors.New("boom")
	l.Has = func(context.Context, effect.Machine, string, []byte) (bool, error) { return false, boom }
	if _, err := l.Plan(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestHomeLinesReadsNothingThroughALinkAtTheFileThatHoldsTheLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/real"], m.Links["/home/ws/f"] = []byte("ws\n"), "/h/real"
	l := homeLinesIn(m, false)
	l.Has = func(context.Context, effect.Machine, string, []byte) (bool, error) {
		t.Error("Has read through a link")
		return true, nil
	}
	_, err := l.Plan(context.Background())
	if !errors.Is(err, ErrLinked) {
		t.Errorf("err = %v, want the link refused", err)
	}
	for _, read := range []string{"read root /h/real", "read home /h/real"} {
		if bytes.Contains([]byte(joinCalls(m)), []byte(read)) {
			t.Errorf("read the target: %v", m.Calls)
		}
	}
}

func TestHomeLinesRefusesALinkOnTheWay(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links["/home/ws"] = "/srv"
	l := homeLinesIn(m, true)
	l.Path = "/home/ws/d/f"
	m.Links["/home/ws/d"] = "/srv/d"
	if _, err := l.Plan(context.Background()); err == nil {
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
	l.M = linkHidden{m, "/home/ws/f"}
	if _, err := l.Plan(context.Background()); err == nil {
		t.Error("Plan read through a link swapped in after the check")
	}
	keep := homeKeep{M: l.M, Path: "/home/ws/f", To: "/home/ws/f.kept~", Home: "/home/ws"}
	if err := keep.Apply(context.Background()); err == nil || len(m.Files["/home/ws/f.kept~"]) != 0 {
		t.Errorf("the keep copied through a link: %v, %q", err, m.Files["/home/ws/f.kept~"])
	}
}

func joinCalls(m *effecttest.Machine) string {
	var b bytes.Buffer
	for _, c := range m.Calls {
		b.WriteString(c + "\n")
	}
	return b.String()
}

func applyAll(t *testing.T, l HomeLines) int {
	cs, err := l.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return len(cs)
}

func TestHomeLinesCreatesAMissingFileOwnedByTheAccount(t *testing.T) {
	m := effecttest.NewMachine()
	l := homeLinesIn(m, true)
	if n := applyAll(t, l); n != 1 {
		t.Fatalf("%d changes", n)
	}
	if string(m.Files["/home/ws/f"]) != "ws\n" || m.Owners["/home/ws/f"] != [2]int{1000, 1001} || m.Modes["/home/ws/f"] != 0o600 {
		t.Errorf("file %q, owner %v, mode %o", m.Files["/home/ws/f"], m.Owners["/home/ws/f"], m.Modes["/home/ws/f"])
	}
	if n := applyAll(t, l); n != 0 {
		t.Errorf("a second plan holds %d changes", n)
	}
}

func TestHomeLinesPlansNothingWhereTheFileHoldsTheLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("a\nws\n")
	if n := applyAll(t, homeLinesIn(m, true)); n != 0 {
		t.Errorf("%d changes", n)
	}
}

func TestHomeLinesRefusesALinkAtTheFileThatLacksTheLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/real"], m.Links["/home/ws/f"] = []byte("a\n"), "/h/real"
	_, err := homeLinesIn(m, true).Plan(context.Background())
	if !errors.Is(err, ErrLinked) || !strings.Contains(err.Error(), "is a link to /h/real") {
		t.Errorf("err %v", err)
	}
}

// With Follow, as on the Mac, the user reads through a link at the file. The fake's file at the
// path stands for that read.
func TestHomeLinesFollowsALinkAtTheFile(t *testing.T) {
	for _, c := range []struct {
		name, text string
		want       error
	}{
		{"holding", "a\nws\n", nil},
		{"lacking", "a\n", ErrLinked},
		{"leading nowhere", "", ErrLinked},
	} {
		m := effecttest.NewMachine()
		m.Links["/home/ws/f"] = "/h/real"
		if c.text != "" {
			m.Files["/home/ws/f"] = []byte(c.text)
		}
		l := homeLinesIn(m, false)
		l.Follow = true
		cs, err := l.Plan(context.Background())
		if !errors.Is(err, c.want) || len(cs) != 0 {
			t.Errorf("%s: plan = %v, %v; want no change and %v", c.name, cs, err, c.want)
		}
	}
}

func TestHomeLinesKeepsAndRewritesAFileThatLacksTheLines(t *testing.T) {
	for _, c := range []struct {
		name, old, want string
		top             bool
	}{
		{"top", "a\n", "ws\n\na\n", true},
		{"end", "a", "a\nws\n", false},
		{"end after a newline", "a\n", "a\nws\n", false},
		{"end of an empty file", "", "ws\n", false},
	} {
		m := effecttest.NewMachine()
		m.Files["/home/ws/f"], m.Modes["/home/ws/f"] = []byte(c.old), 0o640
		m.Owners["/home/ws/f"] = [2]int{1002, 1003}
		l := homeLinesIn(m, c.top)
		if n := applyAll(t, l); n != 2 {
			t.Fatalf("%s: %d changes", c.name, n)
		}
		kept := "/home/ws/f.20260102030405~"
		if string(m.Files[kept]) != c.old || m.Owners[kept] != [2]int{1002, 1003} || m.Modes[kept] != 0o640 {
			t.Errorf("%s: kept %q, %v, %o", c.name, m.Files[kept], m.Owners[kept], m.Modes[kept])
		}
		if string(m.Files["/home/ws/f"]) != c.want || m.Owners["/home/ws/f"] != [2]int{1002, 1003} || m.Modes["/home/ws/f"] != 0o640 {
			t.Errorf("%s: file %q, %v, %o", c.name, m.Files["/home/ws/f"], m.Owners["/home/ws/f"], m.Modes["/home/ws/f"])
		}
		if n := applyAll(t, l); n != 0 {
			t.Errorf("%s: a second plan holds %d changes", c.name, n)
		}
	}
}

func TestHomeLinesShowsOnlyWsLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("secret\n")
	cs, err := homeLinesIn(m, false).Plan(context.Background())
	if err != nil || len(cs) != 2 {
		t.Fatalf("%v, %v", cs, err)
	}
	got := step.ExcerptOf(cs[1])
	if !strings.Contains(got, "+ws") || strings.Contains(got, "secret") {
		t.Errorf("excerpt %q", got)
	}
}

func TestHomeLinesDryRunAndReadShowOnlyWsLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/f"] = []byte("[http]\n\textraHeader = bearer T\n")
	cs, err := homeLinesIn(m, true).Plan(context.Background())
	if err != nil || len(cs) != 2 {
		t.Fatalf("%v, %v", cs, err)
	}
	for name, got := range map[string]string{"String": cs[1].String(), "Pending": step.PendingOf(cs[1]).Text} {
		if strings.Contains(got, "bearer") || strings.Contains(got, "[http]") {
			t.Errorf("%s shows the user's lines: %q", name, got)
		}
	}
	if !strings.Contains(cs[1].String(), "+ws") {
		t.Errorf("String %q, want the lines ws adds", cs[1].String())
	}
	if got := step.Headline(cs[1]); got != "write /home/ws/f" {
		t.Errorf("Headline = %q, want write /home/ws/f", got)
	}
}
