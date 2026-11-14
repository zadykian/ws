package host

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

// hiddenAsk keeps the text of each hidden question, and ends the run.
type hiddenAsk struct{ asked []string }

func (h *hiddenAsk) AskHidden(_ context.Context, text string) (string, error) {
	h.asked = append(h.asked, text)
	return "", errors.New("ended")
}

func TestSudoNamesTheServer(t *testing.T) {
	m := effecttest.NewMachine()
	p := &hiddenAsk{}
	s := &Sudo{SSH: remote.SSH{Host: "lab", M: m, Lock: "/l"}, P: p, Out: &effecttest.Out{}}
	if err := s.ask(context.Background(), "ada"); err == nil || !strings.Contains(err.Error(), "ada's sudo on lab asks for a password") {
		t.Errorf("ask with no terminal = %v; want lab", err)
	}
	s.Tty = true
	_ = s.ask(context.Background(), "ada")
	if len(p.asked) != 1 || p.asked[0] != "ada's sudo password on lab" {
		t.Errorf("asked %q; want the question of lab's sudo", p.asked)
	}
	if got := (sudoAsk{"ada", "lab"}).String(); !strings.HasSuffix(got, "ada's sudo password on lab, at the terminal") {
		t.Errorf("sudoAsk = %q; want lab", got)
	}
	if _, _, err := s.Rooted([]string{"id"}, nil); err == nil || !strings.HasSuffix(err.Error(), "how to reach root on lab") {
		t.Errorf("Rooted before Plan = %v; want lab", err)
	}
}
