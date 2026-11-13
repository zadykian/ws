package brew_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
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
