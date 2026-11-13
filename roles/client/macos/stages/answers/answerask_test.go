package answers

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

func newAsker(p effect.Prompt, env map[string]string) (*asker, *effecttest.Out) {
	out := &effecttest.Out{}
	return &asker{conf: "/h/mac.conf", prog: "ws client macos wizard", p: p, tty: true, out: out, lan: func() string { return "" },
		env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, out
}

func TestAskerTakesTheEnvironmentFirst(t *testing.T) {
	a, out := newAsker(&effecttest.Prompt{T: t}, map[string]string{"WS_REMOTE": "r.example", "WS_RESOLVERS": ""})
	if v, err := a.ask(context.Background(), conf.Question{Key: "WS_REMOTE"}); v != "r.example" || err != nil {
		t.Errorf("ask = %q, %v", v, err)
	}
	if v, err := a.ask(context.Background(), conf.Question{Key: "WS_RESOLVERS"}); v != "" || err != nil || len(out.Lines) != 0 {
		t.Errorf("an empty value set: ask = %q, %v, out %v", v, err, out.Lines)
	}
	if a.lines != "WS_REMOTE=r.example\nWS_RESOLVERS=\n" {
		t.Errorf("lines = %q", a.lines)
	}
}

func TestAskerRefusesABadEnvironment(t *testing.T) {
	a, out := newAsker(&effecttest.Prompt{T: t}, map[string]string{"WS_REMOTE_PORT": "99999"})
	_, err := a.ask(context.Background(), conf.Question{Key: "WS_REMOTE_PORT"})
	if err == nil || !strings.Contains(err.Error(), "WS_REMOTE_PORT comes from the environment") ||
		len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "is not a port") {
		t.Errorf("ask = %v, out %v", err, out.Lines)
	}
}

func TestAskerAsksTheMaintainerAgain(t *testing.T) {
	var w bytes.Buffer
	term := effect.NewTerminal(strings.NewReader("99999\n\n"), &w)
	a, out := newAsker(term, nil)
	q := conf.Question{Key: "WS_REMOTE_PORT", Text: "The VM's SSH port", Def: "22", Need: "needs a port"}
	if v, err := a.ask(context.Background(), q); v != "22" || err != nil {
		t.Fatalf("ask = %q, %v", v, err)
	}
	want := "The VM's SSH port (WS_REMOTE_PORT) [22]: ws client macos wizard: Answers: WS_REMOTE_PORT=99999 is not a port, 1 to 65535\n" +
		"The VM's SSH port (WS_REMOTE_PORT) [22]: "
	if w.String() != want || len(out.Lines) != 1 || !strings.HasPrefix(out.Lines[0], "say: Answers: values for /h/mac.conf;") {
		t.Errorf("terminal = %q, out %v", w.String(), out.Lines)
	}
	if _, err := a.ask(context.Background(), q); err == nil || !strings.Contains(err.Error(), "terminal ended before WS_REMOTE_PORT's answer") {
		t.Errorf("ask at the end of input = %v", err)
	}
}

func TestAskerRefusesWithoutATerminal(t *testing.T) {
	a, out := newAsker(&effecttest.Prompt{T: t}, nil)
	a.tty = false
	_, err := a.ask(context.Background(), conf.Question{Key: "WS_TARGET"})
	if err == nil || err.Error() != "Answers: no terminal to ask WS_TARGET: set WS_TARGET" || len(out.Lines) != 0 {
		t.Errorf("ask with no terminal = %v, saying %q", err, out.Lines)
	}
}

// chooser answers each choice with answer and keeps the questions. It takes no line.
type chooser struct {
	effect.Prompt
	answer string
	asked  []effect.ChoiceQuestion
}

func (c *chooser) Choose(_ context.Context, q effect.ChoiceQuestion) (string, error) {
	c.asked = append(c.asked, q)
	return c.answer, nil
}

func TestAskerChoosesTheTargetOnAScreen(t *testing.T) {
	c := &chooser{Prompt: &effecttest.Prompt{T: t}, answer: "mac-vm"}
	a, _ := newAsker(c, nil)
	q := conf.Question{Key: "WS_TARGET", Text: "The machine ws runs on", Def: "lan", Need: "needs"}
	if v, err := a.ask(context.Background(), q); v != "mac-vm" || err != nil || a.lines != "WS_TARGET=mac-vm\n" {
		t.Fatalf("ask = %q, %v, lines %q", v, err, a.lines)
	}
	want := effect.ChoiceQuestion{Text: "The machine ws runs on (WS_TARGET)", Options: []string{"lan", "cloud", "mac-vm"}, Default: "lan"}
	if len(c.asked) != 1 || c.asked[0].Text != want.Text || !slices.Equal(c.asked[0].Options, want.Options) ||
		c.asked[0].Default != want.Default {
		t.Errorf("asked %+v, want %+v", c.asked, want)
	}
}

func TestAskerAsksALineOfAKeyWithNoOptions(t *testing.T) {
	c := &chooser{Prompt: &effecttest.Prompt{T: t, Answers: map[string]string{"The VM's SSH port (WS_REMOTE_PORT) [22]": ""}}}
	a, _ := newAsker(c, nil)
	q := conf.Question{Key: "WS_REMOTE_PORT", Text: "The VM's SSH port", Def: "22", Need: "needs a port"}
	if v, err := a.ask(context.Background(), q); v != "22" || err != nil || len(c.asked) != 0 {
		t.Errorf("ask = %q, %v, choices %+v", v, err, c.asked)
	}
}
