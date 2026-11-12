package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestGhApply(t *testing.T) {
	for name, tc := range map[string]struct {
		tty  bool
		keys string
		want string // the stop, or a call
	}{
		"login and add":  {true, "ssh-rsa BBBB\n", "run /bin/gh ssh-key add /h/k.pub --title mac"},
		"already listed": {true, "ssh-ed25519 AAAAC3\n", ""},
		"no terminal":    {false, "", "no terminal is there for gh auth login -p https -w"},
		"login failed":   {true, "!", "gh auth login -p https -w -h github.com -s"},
	} {
		g, m := newGh(found)
		g.Tty = tc.tty
		m.Results["/bin/gh auth login -p https -w -h github.com -s "+ghScopes] = effecttest.Result{Err: errors.New("x")}
		if tc.keys != "!" {
			m.Results["/bin/gh auth login -p https -w -h github.com -s "+ghScopes] = effecttest.Result{}
		}
		for _, path := range []string{"user/keys", "user/ssh_signing_keys"} {
			m.Results["/bin/gh api --paginate "+path+" --jq .[].key"] = effecttest.Result{Out: []byte(tc.keys)}
		}
		m.Results["/bin/gh ssh-key add /h/k.pub --title mac"] = effecttest.Result{}
		err := g.Apply(context.Background(), step.Plan{Changes: []step.Change{
			ghLogin{[]string{"login", "-p", "https", "-w"}}, ghAdd{"/h/k.pub", "mac"}}})
		calls := strings.Join(m.Calls, "\n")
		switch {
		case tc.want == "" && (err != nil || strings.Contains(calls, "ssh-key add")),
			tc.want != "" && !strings.Contains(calls+errText(err), tc.want):
			t.Errorf("%s: Apply = %v, calls %q; want %q", name, err, calls, tc.want)
		}
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// The plan after the add finds the key listed: the run that added it says so once, by the add.
func TestGhSaysNoAlreadyAfterItsAdd(t *testing.T) {
	g, m := newGh(found)
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
	for _, path := range []string{"user/keys", "user/ssh_signing_keys"} {
		m.Results["/bin/gh api --paginate "+path+" --jq .[].key"] = effecttest.Result{Out: []byte("ssh-rsa BBBB\n")}
	}
	m.Results["/bin/gh ssh-key add /h/k.pub --title mac"] = effecttest.Result{}
	if err := g.Apply(context.Background(), step.Plan{Changes: []step.Change{ghAdd{"/h/k.pub", "mac"}}}); err != nil {
		t.Fatal(err)
	}
	listed := effecttest.Result{Out: []byte(strings.Join(strings.Fields(pubText)[:2], " ") + "\n")}
	for _, path := range []string{"user/keys", "user/ssh_signing_keys"} {
		m.Results["/bin/gh api --paginate "+path+" --jq .[].key"] = listed
	}
	if p, err := g.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan after the add = %v, %v; want nothing", p.Changes, err)
	}
	want := "say: " + ghStage + ": ran gh ssh-key add /h/k.pub --title mac"
	if got := g.Out.(*effecttest.Out).Lines; len(got) != 1 || got[0] != want {
		t.Errorf("lines = %q, want %q alone", got, want)
	}
}
