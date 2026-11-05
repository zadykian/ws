package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// callsWith returns the calls of m that hold word.
func callsWith(m *effecttest.Machine, word string) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.Contains(c, word) {
			got = append(got, c)
		}
	}
	return got
}

func TestGhLoginTakesTheTerminal(t *testing.T) {
	g, m := newGh(found)
	g.Tty, g.F = true, m
	m.Results["/bin/gh auth login -p https -w -h github.com -s "+ghScopes] = effecttest.Result{}
	if err := g.login(context.Background(), ghLogin{[]string{"login", "-p", "https", "-w"}}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"foreground /bin/gh auth login -p https -w -h github.com -s " + ghScopes}; !slices.Equal(m.Calls[len(m.Calls)-1:], want) {
		t.Errorf("calls = %q, want %q last", m.Calls, want)
	}
}

func TestMacKeyMakeTakesTheTerminal(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_REMOTE=\n")
	k.F = m
	m.Results["ssh-keygen -q -t ed25519 -a 100 -f /k"] = effecttest.Result{}
	m.Results["ssh-add --apple-use-keychain /k"] = effecttest.Result{}
	if err := k.make(context.Background(), "/k"); err != nil {
		t.Fatal(err)
	}
	want := []string{"foreground ssh-keygen -q -t ed25519 -a 100 -f /k", "foreground ssh-add --apple-use-keychain /k"}
	if got := callsWith(m, "foreground"); !slices.Equal(got, want) {
		t.Errorf("foreground calls = %q, want %q", got, want)
	}
	if got := callsWith(m, "pass"); len(got) != 0 {
		t.Errorf("a passphrase's command ran by Pass: %q", got)
	}
}

func TestWsClaudeLoginTakesTheTerminal(t *testing.T) {
	r, m := newRun("lan", "claude login\n", false, false)
	r.Host.SSH.F = m
	c := &WsClaude{Run: r, Tty: true}
	_ = c.Apply(context.Background(), step.Plan{})
	got := callsWith(m, `claude auth login`)
	if len(got) != 1 || !strings.HasPrefix(got[0], "foreground ssh ") {
		t.Errorf("login calls = %q, want one by foreground", got)
	}
}
