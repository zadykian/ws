package devtools

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// lookupOf is the run's env of the secrets task, which holds env alone.
func lookupOf(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

// tokenText is TokenFile as mise writes it: ws's header, a comment, and the token on the last line.
const tokenText = "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
	"# From WS_GITHUB_TOKEN.\ndummy-a\n"

// In the task's run, the token is the input's, which mise writes to the file before any unit and
// in a plan nowhere. Without the input, the file's last line gives it.
func TestRunTokenTakesTheInputOfTheRun(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[TokenFile] = []byte(tokenText)
	for _, c := range []struct {
		lookup func(string) (string, bool)
		want   string
	}{
		{lookupOf(map[string]string{TokenVar: "dummy-b"}), "dummy-b"},
		{lookupOf(map[string]string{TokenVar: ""}), ""},
		{lookupOf(map[string]string{}), "dummy-a"},
		{nil, "dummy-a"},
	} {
		if got, err := RunToken(m, c.lookup); err != nil || got != c.want {
			t.Errorf("want %q: got %q, %v", c.want, got, err)
		}
	}
	if got, err := RunToken(effecttest.NewMachine(), nil); err != nil || got != "" {
		t.Errorf("without the file: got %q, %v", got, err)
	}
}

// gh runs as root from a cleared env and takes the token on its input alone. A login that holds
// the token plans nothing; without one, it stays.
func TestGHLoginLogsInByTheTokenOnItsInput(t *testing.T) {
	for _, u := range []account.User{{Name: "root", Home: "/root"}} {
		gh := "env -i -C /root HOME=/root " + account.SystemPath + " env -u A -u B gh"
		m := effecttest.NewMachine()
		m.Results[gh+" auth token -h github.com"] = effecttest.Result{Out: []byte("dummy-a\n")}
		m.Results[gh+" auth login -h github.com -p https --insecure-storage --with-token"] = effecttest.Result{}
		for token, due := range map[string]int{"dummy-a": 0, "dummy-b": 1, "": 0} {
			m.Calls, m.Inputs = nil, nil
			g := GHLogin{M: m, U: u, Token: token, Hidden: []string{"A", "B"}}
			changes, err := g.Plan(t.Context())
			if err != nil || len(changes) != due {
				t.Fatalf("%s, token %q: plan = %d changes, %v; want %d", u.Name, token, len(changes), err, due)
			}
			if token == "" && len(m.Calls) != 0 {
				t.Errorf("%s: without a token gh ran: %q", u.Name, m.Calls)
			}
			for _, ch := range changes {
				if err := ch.Apply(t.Context()); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(ch.String(), "dummy") {
					t.Errorf("%s: the change %q shows the token", u.Name, ch.String())
				}
			}
			if due == 1 && !slices.Equal(m.Inputs, []string{token + "\n"}) {
				t.Errorf("%s: gh's input %q, want the token", u.Name, m.Inputs)
			}
			if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "dummy") }) {
				t.Errorf("%s: a command's arguments hold the token: %q", u.Name, m.Calls)
			}
		}
	}
}

// A gh that holds no login fails gh auth token, which plans the login.
func TestGHLoginWithoutALoginPlansIt(t *testing.T) {
	m := effecttest.NewMachine()
	g := GHLogin{M: m, U: account.User{Name: "root", Home: "/root"}, Token: "dummy-a"}
	if changes, err := g.Plan(t.Context()); err != nil || len(changes) != 1 {
		t.Errorf("plan = %d changes, %v; want the login", len(changes), err)
	}
}
