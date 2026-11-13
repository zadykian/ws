package cli

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
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
