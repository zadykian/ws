package setup_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	gitHome = "/Users/me"
	gitPub  = gitHome + "/.ssh/id_ed25519_signing.pub"
	gitKey  = "ssh-ed25519 AAAAC3Nz me@mac\n"
)

func gitStep(m *effecttest.Machine, look func(string) (string, error)) *setup.Git {
	m.Results["ssh-keygen -l -f "+gitPub] = effecttest.Result{}
	if look == nil {
		look = func(name string) (string, error) { return "/opt/bin/" + name, nil }
	}
	return &setup.Git{
		Conf: conf.Conf{GitName: "A B", GitEmail: "a@b.c"},
		Home: gitHome, M: m, Out: &effecttest.Out{}, LookPath: look,
	}
}

// prereqs returns the changes that make the key or install gh, which the files wait on.
func prereqs(p step.Plan) []step.Change {
	var cs []step.Change
	for _, c := range p.Changes {
		switch c.(type) {
		case setup.MakeKey, setup.Run:
			cs = append(cs, c)
		}
	}
	return cs
}

func TestGitWithoutNameLeavesGitAlone(t *testing.T) {
	p, err := (&setup.Git{M: effecttest.NewMachine(), Out: &effecttest.Out{}}).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", p.Changes, err)
	}
}

func TestGitPlansGh(t *testing.T) {
	missing := func(n string) (string, error) {
		if n == "gh" {
			return "", errors.New("not found")
		}
		return "/opt/bin/brew", nil
	}
	for name, tc := range map[string]struct {
		look func(string) (string, error)
		want int
		err  string
	}{
		"gh there":    {nil, 0, ""},
		"no gh":       {missing, 1, ""},
		"no brew":     {func(string) (string, error) { return "", errors.New("none") }, 0, "install Homebrew first"},
		"relative gh": {func(string) (string, error) { return "bin/gh", nil }, 0, "drop PATH's relative directories"},
	} {
		m := effecttest.NewMachine()
		m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt\n")}
		m.Files[gitPub], m.Files[gitHome+"/.ssh/id_ed25519_signing"] = []byte(gitKey), []byte("private")
		p, err := gitStep(m, tc.look).Plan(context.Background())
		if (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.err)
		} else if len(prereqs(p)) != tc.want {
			t.Errorf("%s: changes = %v, want %d", name, p.Changes, tc.want)
		}
	}
}

func TestGitApplyInstallsGh(t *testing.T) {
	m, _ := gitReady()
	m.Results["brew install gh"] = effecttest.Result{}
	install := setup.Run{Name: "brew", Args: []string{"install", "gh"}}
	err := gitStep(m, nil).Apply(context.Background(), step.Plan{Changes: []step.Change{install}})
	if err != nil || m.Calls[0] != "run brew install gh" {
		t.Errorf("Apply = %v, calls %q; want brew install gh first", err, m.Calls)
	}
}

func TestGitPlansTheSigningKey(t *testing.T) {
	both := map[string]string{gitPub: gitKey, gitHome + "/.ssh/id_ed25519_signing": "private"}
	for name, tc := range map[string]struct {
		files map[string]string
		want  string
		err   string
	}{
		"there":     {both, "", ""},
		"missing":   {nil, "would make git's signing key, " + gitHome + "/.ssh/id_ed25519_signing", ""},
		"pub alone": {map[string]string{gitPub: gitKey}, "", "without its private key"},
		"two keys": {map[string]string{gitPub: gitKey + gitKey, gitHome + "/.ssh/id_ed25519_signing": "k"}, "",
			"holds no public key alone"},
		"no pub": {map[string]string{gitHome + "/.ssh/id_ed25519_signing": "k"}, "", "is missing beside its key"},
	} {
		m := effecttest.NewMachine()
		for f, text := range tc.files {
			m.Files[f] = []byte(text)
		}
		p, err := gitStep(m, nil).Plan(context.Background())
		if (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.err)
		} else if got := fmt.Sprint(prereqs(p)); (tc.want == "") != (len(prereqs(p)) == 0) || (tc.want != "" && got != "["+tc.want+"]") {
			t.Errorf("%s: changes = %v, want %q", name, p.Changes, tc.want)
		}
	}
}

func TestGitChecksAKeyNamedInMacConf(t *testing.T) {
	m := effecttest.NewMachine()
	g := gitStep(m, nil)
	g.Conf.GitSigningKey = "~/k.pub"
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "WS_GIT_SIGNING_KEY, is missing") {
		t.Errorf("a missing .pub: err = %v", err)
	}
	m.Files[gitHome+"/k.pub"] = []byte(gitKey)
	m.Results["ssh-keygen -l -f "+gitHome+"/k.pub"] = effecttest.Result{Err: errors.New("bad")}
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "ssh-keygen cannot read") {
		t.Errorf("a key ssh-keygen rejects: err = %v", err)
	}
	m.Results["ssh-keygen -l -f "+gitHome+"/k.pub"] = effecttest.Result{}
	if p, err := g.Plan(context.Background()); err != nil || len(prereqs(p)) != 0 {
		t.Errorf("a good key: Plan = %v, %v", p.Changes, err)
	}
}

func TestGitApplyMakesTheKey(t *testing.T) {
	m, _ := gitReady() // the key that ssh-keygen makes is there when Apply plans again
	m.Results["ssh-keygen -q -t ed25519 -N  -f /k"] = effecttest.Result{}
	err := gitStep(m, nil).Apply(context.Background(), step.Plan{Changes: []step.Change{setup.MakeKey{Path: "/k"}}})
	if err != nil || m.Calls[0] != "run ssh-keygen -q -t ed25519 -N  -f /k" {
		t.Errorf("Apply = %v, calls %q; want ssh-keygen first", err, m.Calls)
	}
}

// gitReady is a Mac with the key and gh there.
func gitReady() (*effecttest.Machine, *setup.Git) {
	m := effecttest.NewMachine()
	m.Files[gitPub], m.Files[gitHome+"/.ssh/id_ed25519_signing"] = []byte(gitKey), []byte("private")
	m.Dirs[gitHome+"/.config"], m.Dirs[gitHome+"/.config/ws"] = 0o755, 0o700
	return m, gitStep(m, nil)
}

func TestGitWritesGitconfig(t *testing.T) {
	m, g := gitReady()
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan = %v, %v; want one write", p.Changes, err)
	}
	if err := g.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	dir := gitHome + "/.config/ws/"
	if got := string(m.Files[dir+"gitconfig"]); !strings.Contains(got, "signingkey = \""+gitPub+"\"") ||
		!strings.Contains(got, "helper = !/opt/bin/gh auth git-credential") {
		t.Errorf("gitconfig = %q", got)
	}
	if again, err := g.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("Plan after Apply = %v, %v; want nothing", again.Changes, err)
	}
}

func TestGitFixesAModeAndMakesDirectories(t *testing.T) {
	m, g := gitReady()
	delete(m.Dirs, gitHome+"/.config/ws")
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 2 || p.Changes[0].String() != "would make "+gitHome+"/.config/ws, mode 700" {
		t.Fatalf("Plan = %v, %v; want the directory and a write", p.Changes, err)
	}
	m, g = gitReady()
	if err := g.Apply(context.Background(), mustPlan(t, g)); err != nil {
		t.Fatal(err)
	}
	m.Modes[gitHome+"/.config/ws/gitconfig"] = 0o600
	if p, err := g.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Errorf("Plan with a wrong mode = %v, %v; want one change", p.Changes, err)
	}
}

func mustPlan(t *testing.T, g *setup.Git) step.Plan {
	p, err := g.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
