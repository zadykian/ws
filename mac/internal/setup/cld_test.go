package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

const (
	cldBin  = "/h/.local/bin/cld"
	cldLine = "would add the lines that start cld's watcher to config.d/ws, where that cld has restore tabs"
	cldNo   = "no"
)

func cldStep(check bool) (*setup.Cld, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Modes[cldBin] = 0o755
	m.Files[cldBin] = nil
	m.Results[cldBin+" --version"] = effecttest.Result{Out: []byte("cld 0.13.0\n")}
	m.Results[cldBin+" help update"] = effecttest.Result{}
	m.Results[cldBin+" help restore tabs"] = effecttest.Result{Err: errors.New(cldNo)}
	m.Results["brew list --cask ghostty"] = effecttest.Result{Err: errors.New(cldNo)}
	return &setup.Cld{Home: "/h", M: m, Out: &effecttest.Out{}, Check: check}, m
}

func cldChanges(t *testing.T, s *setup.Cld) []string {
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
	s, m := cldStep(true)
	delete(m.Files, cldBin)
	got := strings.Join(cldChanges(t, s), "\n")
	want := "would install cld into ~/.local/bin with the install.sh of its latest release\n" + cldLine
	if got != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestCldRunInstallsWithoutTheNote(t *testing.T) {
	s, m := cldStep(false)
	delete(m.Files, cldBin)
	if got := cldChanges(t, s); len(got) != 1 {
		t.Errorf("Plan = %q; want the install alone", got)
	}
}
