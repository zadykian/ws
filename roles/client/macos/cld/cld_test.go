package cld_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/client/macos/cld"
)

const (
	cldLatest = "curl -fsSIL --retry 3 -o /dev/null -w %{url_effective} " +
		"https://github.com/zadykian/cld/releases/latest"
	cldBin  = "/h/.local/bin/cld"
	cldLine = "would add the lines that start cld's watcher to config.d/ws, where that cld has restore tabs"
	cldNo   = "no"
)

func cldStep(check bool) (resource.Lazy, *effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Modes[cldBin] = 0o755
	m.Files[cldBin] = nil
	m.Results[cldBin+" --version"] = effecttest.Result{Out: []byte("cld 0.13.0\n")}
	m.Results[cldBin+" help update"] = effecttest.Result{}
	m.Results[cldBin+" help restore tabs"] = effecttest.Result{Err: errors.New(cldNo)}
	m.Results["brew list --cask ghostty"] = effecttest.Result{Err: errors.New(cldNo)}
	return (&cld.Cld{Home: "/h", M: m, Out: out, Check: check}).Step(), m, out
}

func cldChanges(t *testing.T, s resource.Lazy) []string {
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.String())
	}
	return got
}

func TestCldMissingIsInstalled(t *testing.T) {
	s, m, _ := cldStep(true)
	delete(m.Files, cldBin)
	got := strings.Join(cldChanges(t, s), "\n")
	want := "would install cld into ~/.local/bin with the install.sh of its latest release\n" + cldLine
	if got != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestCldRunInstallsWithoutTheNote(t *testing.T) {
	s, m, _ := cldStep(false)
	delete(m.Files, cldBin)
	if got := cldChanges(t, s); len(got) != 1 {
		t.Errorf("Plan = %q; want the install alone", got)
	}
}
