package gh

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
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
	want := "say: " + stage.GitHub + ": ran gh ssh-key add /h/k.pub --title mac"
	if got := g.Out.(*effecttest.Out).Lines; len(got) != 1 || got[0] != want {
		t.Errorf("lines = %q, want %q alone", got, want)
	}
}

// brewAlone finds Homebrew on PATH, and no gh.
func brewAlone(name string) (string, error) {
	if name == "brew" {
		return "/bin/brew", nil
	}
	return "", errors.New("not found")
}

// Where PATH has no gh, the plan holds its install, and the login needs gh there. So Apply runs
// the install first, and stops at a failed one before it logs in.
func TestGhApplyInstallsGhBeforeTheLogin(t *testing.T) {
	const gh = "/opt/homebrew/bin/gh"
	const login = gh + " auth login -p https -w -h github.com -s " + ghScopes
	for name, tc := range map[string]struct {
		brew error
		want string // the stop, or empty where the login runs
	}{
		"installed":   {nil, ""},
		"brew failed": {errors.New("exit status 1"), stage.GitHub + ": brew install gh failed"},
	} {
		g, m := newGh(brewAlone)
		g.Tty = true
		m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt/homebrew\n")}
		m.Results["brew install gh"] = effecttest.Result{Err: tc.brew}
		m.Results[login] = effecttest.Result{}
		for _, path := range []string{"user/keys", "user/ssh_signing_keys"} {
			m.Results[gh+" api --paginate "+path+" --jq .[].key"] = effecttest.Result{}
		}
		m.Results[gh+" ssh-key add /h/k.pub --title mac"] = effecttest.Result{}
		p, err := g.Plan(context.Background())
		if err != nil {
			t.Fatalf("%s: Plan = %v", name, err)
		}
		err = g.Apply(context.Background(), p)
		installs, at, logins := 0, -1, -1
		for i, call := range m.Calls {
			switch call {
			case "run brew install gh":
				installs, at = installs+1, i
			case "run " + login:
				logins = i
			}
		}
		if installs != 1 || errText(err) != tc.want || (tc.want == "") != (logins > at) {
			t.Errorf("%s: Apply = %v, calls %q; want one install, then %q", name, err, m.Calls, tc.want)
		}
	}
}
