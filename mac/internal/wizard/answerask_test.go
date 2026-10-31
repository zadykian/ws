package wizard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func newAsker(env map[string]string) (*asker, *effecttest.Out) {
	out := &effecttest.Out{}
	return &asker{out: out, lan: func() string { return "" },
		env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, out
}

func TestAskerTakesTheEnvironmentFirst(t *testing.T) {
	a, out := newAsker(map[string]string{"WS_REMOTE": "r.example", "WS_HOOK": ""})
	if v, err := a.ask(question{key: "WS_REMOTE"}); v != "r.example" || err != nil {
		t.Errorf("ask = %q, %v", v, err)
	}
	if v, err := a.ask(question{key: "WS_HOOK"}); v != "" || err != nil || len(out.Lines) != 0 {
		t.Errorf("an empty value set: ask = %q, %v, out %v", v, err, out.Lines)
	}
	if a.lines != "WS_REMOTE=r.example\nWS_HOOK=\n" {
		t.Errorf("lines = %q", a.lines)
	}
}

func TestAskerRefusesABadEnvironment(t *testing.T) {
	a, out := newAsker(map[string]string{"WS_REMOTE_PORT": "99999"})
	_, err := a.ask(question{key: "WS_REMOTE_PORT"})
	if err == nil || !strings.Contains(err.Error(), "WS_REMOTE_PORT comes from the environment") ||
		len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "is not a port") {
		t.Errorf("ask = %v, out %v", err, out.Lines)
	}
}
