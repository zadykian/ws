package wizard

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func newAsker(p effect.Prompt, env map[string]string) (*asker, *effecttest.Out) {
	out := &effecttest.Out{}
	return &asker{conf: "/h/mac.conf", prog: "ws mac wizard", p: p, tty: true, out: out, lan: func() string { return "" },
		env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, out
}

func TestAskerTakesTheEnvironmentFirst(t *testing.T) {
	a, out := newAsker(&effecttest.Prompt{T: t}, map[string]string{"WS_REMOTE": "r.example", "WS_HOOK": ""})
	if v, err := a.ask(context.Background(), question{key: "WS_REMOTE"}); v != "r.example" || err != nil {
		t.Errorf("ask = %q, %v", v, err)
	}
	if v, err := a.ask(context.Background(), question{key: "WS_HOOK"}); v != "" || err != nil || len(out.Lines) != 0 {
		t.Errorf("an empty value set: ask = %q, %v, out %v", v, err, out.Lines)
	}
	if a.lines != "WS_REMOTE=r.example\nWS_HOOK=\n" {
		t.Errorf("lines = %q", a.lines)
	}
}

func TestAskerRefusesABadEnvironment(t *testing.T) {
	a, out := newAsker(&effecttest.Prompt{T: t}, map[string]string{"WS_REMOTE_PORT": "99999"})
	_, err := a.ask(context.Background(), question{key: "WS_REMOTE_PORT"})
	if err == nil || !strings.Contains(err.Error(), "WS_REMOTE_PORT comes from the environment") ||
		len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "is not a port") {
		t.Errorf("ask = %v, out %v", err, out.Lines)
	}
}

func TestAskerAsksTheMaintainerAgain(t *testing.T) {
	var w bytes.Buffer
	term := effect.NewTerminal(strings.NewReader("99999\n\n"), &w)
	a, out := newAsker(term, nil)
	q := question{key: "WS_REMOTE_PORT", text: "The VM's SSH port", def: "22", need: "needs a port"}
	if v, err := a.ask(context.Background(), q); v != "22" || err != nil {
		t.Fatalf("ask = %q, %v", v, err)
	}
	want := "The VM's SSH port (WS_REMOTE_PORT) [22]: ws mac wizard: Answers: WS_REMOTE_PORT=99999 is not a port, 1 to 65535\n" +
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
	_, err := a.ask(context.Background(), question{key: "WS_TARGET"})
	if err == nil || err.Error() != "Answers: no terminal to ask WS_TARGET: set WS_TARGET" || len(out.Lines) != 0 {
		t.Errorf("ask with no terminal = %v, saying %q", err, out.Lines)
	}
}
