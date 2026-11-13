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

// appAndFont is the app's cask and the font's part, which waits on the app.
func appAndFont(m *effecttest.Machine, out *effecttest.Out, there bool) (*brew.App, resource.Resource) {
	look := func(string) (string, error) { return "brew", nil }
	failed := map[string]bool{}
	app := &brew.App{Cask: brew.Cask{M: m, Out: out, LookPath: look, Failed: failed, Name: "zed",
		There: there, Missing: "no Zed.app", Holds: true}}
	font := brew.Cask{M: m, Out: out, LookPath: look, Failed: failed, Name: "font", Missing: "no font"}
	return app, app.Then(font)
}

func planned(parts ...resource.Resource) ([][]resource.Change, error) {
	var all [][]resource.Change
	for _, p := range parts {
		cs, err := p.Plan(context.Background())
		if err != nil {
			return all, err
		}
		all = append(all, cs)
	}
	return all, nil
}

func TestThenPlansWhereTheAppIsThereOrComes(t *testing.T) {
	for _, there := range []bool{true, false} {
		m := effecttest.NewMachine()
		app, font := appAndFont(m, &effecttest.Out{}, there)
		m.Results["brew list --cask zed"] = effecttest.Result{Err: errors.New("no")}
		m.Results["brew list --cask font"] = effecttest.Result{Err: errors.New("no")}
		got, err := planned(app, font)
		if err != nil || len(got[1]) != 1 {
			t.Errorf("there %v: plans %v, %v; want the font's install", there, got, err)
		}
	}
}

func TestThenPlansNothingWhereTheAppFailedOrBrewHasItAlone(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	app, font := appAndFont(m, out, false)
	app.Failed["zed"] = true
	if got, err := planned(app, font); err != nil || len(got[0])+len(got[1]) != 0 {
		t.Errorf("failed app: plans %v, %v; want none", got, err)
	}
	app.Failed["zed"] = false
	m.Results["brew list --cask zed"] = effecttest.Result{}
	if got, err := planned(app, font); err != nil || len(got[0])+len(got[1]) != 0 || len(out.Lines) != 1 {
		t.Errorf("gone app: plans %v, %v, said %q; want one warning and none", got, err, out.Lines)
	}
}

func TestThenStopsWhereTheAppStopsWithoutHomebrew(t *testing.T) {
	m := effecttest.NewMachine()
	app, font := appAndFont(m, &effecttest.Out{}, false)
	app.LookPath = func(string) (string, error) { return "", errors.New("none") }
	if _, err := planned(app, font); err == nil {
		t.Error("want the error of the missing Homebrew")
	}
}

func TestThenReportsWhereTheAppIsMissing(t *testing.T) {
	m := effecttest.NewMachine()
	_, font := appAndFont(m, &effecttest.Out{}, false)
	items, err := font.(step.Reporter).Report(context.Background())
	if err != nil || len(items) != 1 || items[0].Name != "font" || items[0].Found != "missing" {
		t.Errorf("Report = %+v, %v; want the font's item", items, err)
	}
}
