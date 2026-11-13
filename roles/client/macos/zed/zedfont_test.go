package zed_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// The font waits on the app. Homebrew has the app's cask, the app is gone and the font is missing:
// the run warns once and plans no font install.
func TestZedFontWaitsOnTheApp(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
	delete(m.Files, "/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf")
	m.Results["brew list --cask zed"] = effecttest.Result{}
	m.Results["brew list --cask font-jetbrains-mono-nerd-font"] = effecttest.Result{Err: errors.New("no")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("Plan = %v, %v, out %v; want one warning and no change", p.Changes, err, out.Lines)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "install --cask font") }) {
		t.Errorf("calls %v; want no install of the font", m.Calls)
	}
}
