package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func newGh(look func(string) (string, error)) (*Gh, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files[confPathForTest] = []byte("WS_IDENTITY=~/k.pub\n")
	m.Files["/h/k.pub"] = []byte(pubText)
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["hostname -s"] = effecttest.Result{Out: []byte("mac\n")}
	return &Gh{GhCLI: GhCLI{Conf: confPathForTest, Home: "/h", LookPath: look, M: m, Out: &effecttest.Out{}}}, m
}

func found(name string) (string, error) { return "/bin/" + name, nil }

func TestGhPlan(t *testing.T) {
	fields := strings.Fields(pubText)
	line := fields[0] + " " + fields[1]
	api := func(m *effecttest.Machine, path string, r effecttest.Result) {
		m.Results["/bin/gh api --paginate "+path+" --jq .[].key"] = r
	}
	ok := effecttest.Result{Out: []byte("ssh-rsa BBBB\n" + line + "\n")}
	none := effecttest.Result{Out: []byte("ssh-rsa BBBB\n")}
	missing := effecttest.Result{Err: errors.New("gh: HTTP 404")}
	for name, tc := range map[string]struct {
		login      bool
		keys, sign effecttest.Result
		want       string
	}{
		"done":       {true, ok, none, ""},
		"no login":   {false, ok, ok, "gh auth login -p https -w -h github.com -s write:public_key,write:ssh_signing_key"},
		"not listed": {true, none, none, "would run gh ssh-key add /h/k.pub --title mac"},
		"no scope":   {true, ok, missing, "gh auth refresh -h github.com"},
		"no keys":    {true, missing, ok, "gh auth refresh -h github.com"},
	} {
		g, m := newGh(found)
		if tc.login {
			m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
		} else {
			m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{Err: errors.New("no login")}
		}
		api(m, "user/keys", tc.keys)
		api(m, "user/ssh_signing_keys", tc.sign)
		p, err := g.Plan(context.Background())
		got := fmt.Sprint(p.Changes)
		if err != nil || (tc.want == "") != (len(p.Changes) == 0) || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

func TestGhPlanInstall(t *testing.T) {
	g, m := newGh(func(n string) (string, error) {
		if n == "brew" {
			return "/bin/brew", nil
		}
		return "", errors.New("not found")
	})
	m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt/homebrew\n")}
	p, err := g.Plan(context.Background())
	got := fmt.Sprint(p.Changes)
	if err != nil || !strings.Contains(got, "brew install gh") || !strings.Contains(got, "gh auth login") ||
		!strings.Contains(got, "gh ssh-key add /h/k.pub --title mac") {
		t.Errorf("Plan = %q, %v", got, err)
	}
}

func TestGhPlanOtherError(t *testing.T) {
	g, m := newGh(found)
	m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
	m.Results["/bin/gh api --paginate user/keys --jq .[].key"] = effecttest.Result{Err: errors.New("HTTP 500")}
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "gh api user/keys failed") {
		t.Errorf("Plan = %v", err)
	}
}

func TestGhPlanMissingKey(t *testing.T) {
	for name, login := range map[string]bool{"login": true, "no login": false} {
		g, m := newGh(found)
		delete(m.Files, "/h/k.pub")
		if login {
			m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{}
		} else {
			m.Results["/bin/gh auth status -h github.com"] = effecttest.Result{Err: errors.New("no login")}
		}
		p, err := g.Plan(context.Background())
		got := fmt.Sprint(p.Changes)
		want := "gh on the Mac: would run gh ssh-key add for the Mac's key, once it is made"
		if err != nil || !strings.HasSuffix(got, want+"]") || strings.Contains(got, "gh auth login") != !login {
			t.Errorf("%s: Plan = %q, %v; want the later add, and a login without one", name, got, err)
		}
		if err := g.Apply(context.Background(), p); err == nil {
			t.Errorf("%s: Apply = nil; want the key's absence", name)
		}
	}
}

// The GitHub stage adds ws's copy of the agent's key, which the Mac key stage wrote.
func TestGhKeyOfTheAgent(t *testing.T) {
	g, m := newGh(found)
	m.Files[confPathForTest] = nil
	agent(m, p256)
	pub, key, err := g.key(context.Background())
	if err != nil || pub != copyOf(p256) || key != strings.TrimSuffix(strings.TrimSuffix(p256, "\n"), " me") {
		t.Errorf("key = %q, %q, %v", pub, key, err)
	}
}
