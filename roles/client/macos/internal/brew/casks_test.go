package brew_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/brew"
)

func casks(m *effecttest.Machine, out *effecttest.Out) brew.Casks {
	look := func(string) (string, error) { return "brew", nil }
	return brew.Casks{M: m, Out: out, Failed: map[string]bool{}, App: "zed", LookPath: look}
}

func TestACaskPlansItsInstallAsAWarnedCommand(t *testing.T) {
	b := casks(effecttest.NewMachine(), &effecttest.Out{})
	app, _, _ := b.Cask(context.Background(), "zed", false, "gone")
	font, _, _ := b.Cask(context.Background(), "font", false, "gone")
	if len(app) != 1 || len(font) != 1 {
		t.Fatalf("Cask = %v, %v; want an install each", app, font)
	}
	if step.Headline(app[0]) != "would run brew install --cask zed" || step.KindOf(app[0]) != step.Install ||
		!app[0].(resource.Warned).Warned() || font[0].(resource.Warned).Warned() {
		t.Errorf("%v: want brew's install, which holds for the app alone", app)
	}
}

func TestAFailedInstallWarnsAndMarksTheCask(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["brew install --cask zed"] = effecttest.Result{Err: errors.New("down")}
	b := casks(m, out)
	cs, _, _ := b.Cask(context.Background(), "zed", false, "gone")
	err := cs[0].(resource.Change).Apply(context.Background())
	if err == nil || !b.Failed["zed"] || len(out.Lines) != 2 || out.Lines[1] != "warn: brew install --cask zed failed" {
		t.Errorf("Apply = %v, failed %v, said %q; want the cause, marked, warned", err, b.Failed, out.Lines)
	}
}

func TestACaskThatIsThereOrFailedPlansNothing(t *testing.T) {
	m := effecttest.NewMachine()
	for _, c := range []brew.Cask{{M: m, Name: "zed", There: true}, {M: m, Name: "zed", Failed: map[string]bool{"zed": true}}} {
		if cs, err := c.Plan(context.Background()); err != nil || len(cs) != 0 {
			t.Errorf("%+v: Plan = %v, %v; want none", c, cs, err)
		}
	}
	if len(m.Calls) != 0 {
		t.Errorf("ran %q; want no brew", m.Calls)
	}
}

func TestACaskStopsWithoutHomebrewAndWarnsWhereBrewHasIt(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	c := brew.Cask{M: m, Out: out, Name: "zed", Missing: "no Zed.app is there",
		LookPath: func(string) (string, error) { return "", errors.New("none") }}
	if _, err := c.Plan(context.Background()); err == nil {
		t.Error("Plan without Homebrew: want an error")
	}
	m.Results["brew list --cask zed"] = effecttest.Result{}
	c.LookPath = func(string) (string, error) { return "brew", nil }
	if cs, err := c.Plan(context.Background()); err != nil || len(cs) != 0 || len(out.Lines) != 1 {
		t.Errorf("Plan = %v, %v, said %q; want a warning alone", cs, err, out.Lines)
	}
}

func TestACaskReportsHowItStands(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["brew list --cask zed"] = effecttest.Result{}
	there, _ := brew.Cask{M: m, Name: "zed", There: true}.Report(context.Background())
	gone, _ := brew.Cask{M: m, Name: "zed", Missing: "no Zed.app is there"}.Report(context.Background())
	if there[0].Found != "installed" || gone[0].Found != "missing" || gone[0].Conflict == "" {
		t.Errorf("reports %+v, %+v; want installed, then missing against Homebrew", there, gone)
	}
}
