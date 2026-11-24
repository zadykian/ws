package cld_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/cld"
)

const (
	cldLatest = "curl -fsSIL --retry 3 -o /dev/null -w %{url_effective} " +
		"https://github.com/zadykian/cld/releases/latest"
	cldBin = "/h/.local/bin/cld"
	cldNo  = "no"
)

// cldOf is cld 0.13.0 in /h's ~/.local/bin, with cld update.
func cldOf() (*cld.Cld, *effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Modes[cldBin] = 0o755
	m.Files[cldBin] = nil
	m.Results[cldBin+" --version"] = effecttest.Result{Out: []byte("cld 0.13.0\n")}
	m.Results[cldBin+" help update"] = effecttest.Result{}
	return &cld.Cld{Home: "/h", M: m, Out: out}, m, out
}

// cldChanges returns the headlines of c's plan.
func cldChanges(t *testing.T, c *cld.Cld) []string {
	cs, err := c.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, ch := range cs {
		got = append(got, ch.String())
	}
	return got
}

func TestCldMissingIsInstalled(t *testing.T) {
	c, m, _ := cldOf()
	delete(m.Files, cldBin)
	want := "would install cld into ~/.local/bin with the install.sh of its latest release"
	if got := cldChanges(t, c); len(got) != 1 || got[0] != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestCldBehindTheLatestIsUpdated(t *testing.T) {
	c, m, _ := cldOf()
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.14.0")}
	want := "would run cld update: cld 0.13.0, the latest 0.14.0"
	if got := cldChanges(t, c); len(got) != 1 || got[0] != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestCldAtTheLatestPlansNothing(t *testing.T) {
	c, m, _ := cldOf()
	m.Results[cldLatest] = effecttest.Result{Out: []byte("https://github.com/zadykian/cld/releases/tag/v0.13.0")}
	if got := cldChanges(t, c); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}

func TestCldWarnsWhereItsVersionFailsOrIsNoRelease(t *testing.T) {
	for out, warns := range map[string]int{"": 1, "cld dev\n": 0} {
		c, m, o := cldOf()
		m.Results[cldBin+" --version"] = effecttest.Result{Out: []byte(out)}
		if out == "" {
			m.Results[cldBin+" --version"] = effecttest.Result{Err: errors.New(cldNo)}
		}
		if got := cldChanges(t, c); len(got) != 0 || len(o.Lines) != warns {
			t.Errorf("version %q: Plan = %q, out %q; want none and %d warnings", out, got, o.Lines, warns)
		}
	}
}

func TestCldWithoutUpdateIsReplacedByTheInstall(t *testing.T) {
	c, m, _ := cldOf()
	m.Results[cldBin+" help update"] = effecttest.Result{Err: errors.New(cldNo)}
	if got := cldChanges(t, c); len(got) != 1 || !strings.HasPrefix(got[0], "would install cld") {
		t.Errorf("Plan = %q; want the install", got)
	}
}
