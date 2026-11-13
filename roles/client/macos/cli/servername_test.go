package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// replies is a prompt that gives each answer in turn and keeps the questions.
type replies struct {
	give []string
	got  []effect.Question
}

func (a *replies) Ask(_ context.Context, q effect.Question) (string, error) {
	a.got = append(a.got, q)
	next := a.give[0]
	a.give = a.give[1:]
	return next, nil
}

func TestAskServerOffersWsAndSetsTheName(t *testing.T) {
	var name string
	ask := &replies{give: []string{"box"}}
	if _, err := (&askServer{ask: ask, tty: true, name: &name}).Plan(t.Context()); err != nil {
		t.Fatal(err)
	}
	q := ask.got[0]
	if name != "box" || q.Text != "The server's name" || q.Default != conf.FirstServer {
		t.Errorf("name = %q, question = %+v; want box, the server's name with ws offered", name, q)
	}
	if err := q.Valid("Lab"); err == nil {
		t.Error("Valid(Lab) passed")
	}
	for _, bad := range []string{"", "1a", "abcdefghijklm", "a_b"} {
		if q.Valid(bad) == nil {
			t.Errorf("Valid(%q) passed", bad)
		}
	}
}

func TestAskServerRefusesWithNoTerminal(t *testing.T) {
	ask := &replies{}
	_, err := (&askServer{ask: ask, name: new(string)}).Plan(t.Context())
	if err == nil || err.Error() != "no terminal to ask the server's name: pass --server NAME" || len(ask.got) != 0 {
		t.Errorf("err = %v, questions = %v", err, ask.got)
	}
}

func TestWizardAsksTheServersNameAndAsksAgainOnABadOne(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	p.TTY = true
	status, _, stderr := runWs(p, "Lab\nbox\n", "client", "macos", "wizard", "--check")
	if status != 1 || !strings.Contains(stderr, "The server's name [ws]: not a server's name: "+conf.NameForm+"\n") ||
		!strings.Contains(stderr, "The server's name [ws]: ") || strings.Contains(stderr, "no terminal to ask the server's name") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if !strings.Contains(strings.Join(p.Machine.(*effecttest.Machine).Calls, "\n"), "read /.config/ws/servers/box/server.conf") {
		t.Errorf("calls = %v; want the stages on box", p.Machine.(*effecttest.Machine).Calls)
	}
}

func TestWizardTakesWsWithAnEmptyAnswer(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	p.TTY = true
	runWs(p, "\n", "client", "macos", "wizard", "--check")
	if !strings.Contains(strings.Join(p.Machine.(*effecttest.Machine).Calls, "\n"), "read /.config/ws/servers/ws/server.conf") {
		t.Errorf("calls = %v; want the stages on ws", p.Machine.(*effecttest.Machine).Calls)
	}
}

func TestWizardRefusesToAskWithNoTerminal(t *testing.T) {
	p, _ := testWizardProcess("darwin", 501)
	status, _, stderr := runWs(p, "", "client", "macos", "wizard", "--check")
	want := "ws client macos wizard: no terminal to ask the server's name: pass --server NAME\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}

func TestWizardAsksNothingWhereAServerExistsOrIsNamed(t *testing.T) {
	for name, args := range map[string][]string{"named": {"--server", "lab"}, "one server": nil} {
		p, _ := testWizardProcess("darwin", 501)
		p.TTY = true
		if name == "one server" {
			m := p.Machine.(*effecttest.Machine)
			m.Files[conf.ServerPath("/.config/ws/mac.conf", "box")] = nil
			withServerDirs(m)
		}
		_, _, stderr := runWs(p, "", append([]string{"client", "macos", "wizard", "--check"}, args...)...)
		if strings.Contains(stderr, "The server's name") {
			t.Errorf("%s: stderr = %q; want no question", name, stderr)
		}
	}
}
