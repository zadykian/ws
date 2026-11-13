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
