package setup_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const gitAdd = "gh ssh-key add " + gitPub + " --type signing --title mac"

func exitFour(t *testing.T) error {
	err := exec.Command("sh", "-c", "exit 4").Run()
	if err == nil {
		t.Fatal("sh -c 'exit 4' succeeded")
	}
	return err
}

func TestGitAddsTheKeyToGh(t *testing.T) {
	_, g := gitReady()
	g.M.(*effecttest.Machine).Results[gitList] = effecttest.Result{}
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 || p.Changes[2].String() != "would run "+gitAdd {
		t.Fatalf("Plan = %v, %v; want two writes and the add", p.Changes, err)
	}
}

func TestGitWaitsForGhBeforeAddingTheKey(t *testing.T) {
	m, g := gitReady()
	g.LookPath = func(n string) (string, error) {
		if n == "gh" {
			return "", errors.New("none")
		}
		return "/opt/bin/" + n, nil
	}
	m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt\n")}
	p, err := g.Plan(context.Background())
	last := p.Changes[len(p.Changes)-1].String()
	if err != nil || !strings.HasSuffix(last, gitAdd+" once gh is installed, where it has a login that lacks the key") {
		t.Errorf("Plan = %v, %v; want the wait on gh last", p.Changes, err)
	}
}

func TestGitWarnsWhereGhCannotRegister(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want string
	}{
		"no login": {exitFour(t), "gh has no login"},
		"other":    {errors.New("boom"), "gh failed: boom"},
	} {
		m, g := gitReady()
		out := g.Out.(*effecttest.Out)
		m.Results[gitList] = effecttest.Result{Err: tc.err}
		p, err := g.Plan(context.Background())
		if err != nil || len(p.Changes) != 3 || len(out.Lines) != 0 {
			t.Fatalf("%s: Plan = %v, %v, lines %q; want two writes and a warning, said later", name, p.Changes, err, out.Lines)
		}
		if w := p.Changes[2].String(); !strings.Contains(w, tc.want) || !strings.Contains(w, "unverified") {
			t.Errorf("%s: last change = %q; want the warning", name, w)
		}
		if err := g.Apply(context.Background(), p); err != nil {
			t.Fatal(err)
		}
		if n := len(out.Lines); n != 3 || !strings.Contains(out.Lines[2], tc.want) {
			t.Errorf("%s: lines = %q; want the warning after the two writes", name, out.Lines)
		}
	}
}

func TestGitChecksThatGhListsTheKeyItAdded(t *testing.T) {
	m, g := gitReady()
	out := g.Out.(*effecttest.Out)
	m.Results[gitList] = effecttest.Result{}
	m.Results["/opt/bin/gh ssh-key add "+gitPub+" --type signing --title mac"] = effecttest.Result{}
	if err := g.Apply(context.Background(), step.Plan{Changes: []step.Change{setup.Register{
		Gh: "/opt/bin/gh", Pub: gitPub, Public: "ssh-ed25519 AAAAC3Nz", Title: "mac"}}}); err != nil {
		t.Fatal(err)
	}
	if len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "lacks the key after "+gitAdd) {
		t.Errorf("lines = %q; want a warning that the list lacks the key", out.Lines)
	}
}

func TestGitGoesOnWhereGhCannotAddTheKey(t *testing.T) {
	m, g := gitReady()
	out := g.Out.(*effecttest.Out)
	m.Results[gitList] = effecttest.Result{}
	m.Results["/opt/bin/gh ssh-key add "+gitPub+" --type signing --title mac"] = effecttest.Result{Err: errors.New("boom")}
	if _, err := step.Run(context.Background(), []step.Step{g}, out, false); err != nil {
		t.Errorf("Run = %v; want the step to warn and go on", err)
	}
	if n := strings.Count(strings.Join(out.Lines, "\n"), "unverified"); n != 1 {
		t.Errorf("lines = %q; want one warning", out.Lines)
	}
}
