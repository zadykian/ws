package ghostty_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// freshGhostty is a Mac with neither the app nor the font, where Homebrew has neither cask.
func freshGhostty() (*effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	for _, name := range []string{"ghostty", "font-jetbrains-mono-nerd-font"} {
		m.Results["brew list --cask "+name] = effecttest.Result{Err: errors.New("no")}
		m.Results["brew install --cask "+name] = effecttest.Result{}
	}
	return m, out
}

// Apply and --check say the casks first, then the files, in this order.
func TestGhosttyPlansAndAppliesInOrder(t *testing.T) {
	m, out := freshGhostty()
	s := ghosttyStep(m, out)
	delete(m.Dirs, "/h/Applications/Ghostty.app")
	delete(m.Files, "/h/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf")
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var heads []string
	for _, c := range p.Changes {
		heads = append(heads, step.Headline(c))
	}
	want := []string{
		"would run brew install --cask ghostty",
		"would run brew install --cask font-jetbrains-mono-nerd-font",
	}
	if len(heads) != 5 || !slices.Equal(heads[:2], want) {
		t.Fatalf("headlines %q; want the casks first, then three files", heads)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(out.Lines[:2], []string{"say: installing ghostty", "say: installing font-jetbrains-mono-nerd-font"}) {
		t.Errorf("said %q; want the casks installed first", out.Lines)
	}
}

// A report lists the app, the font, then the files.
func TestGhosttyReportsInOrder(t *testing.T) {
	m, out := freshGhostty()
	items, err := ghosttyStep(m, out).Report(context.Background())
	var names []string
	for _, it := range items {
		names = append(names, it.Name)
	}
	want := []string{"ghostty", "font-jetbrains-mono-nerd-font", ghosttyDir, ghosttyDir + "/ws.ghostty", ghosttyDir + "/config.ghostty"}
	if err != nil || !slices.Equal(names, want) {
		t.Errorf("items %q, %v; want %q", names, err, want)
	}
}

// The step has three parts: the app's cask, the font's, then the files. The waits are tested apart.
func TestGhosttyPartsAreTheAppTheFontAndTheFiles(t *testing.T) {
	m, out := freshGhostty()
	parts, err := ghosttyStep(m, out).Parts(context.Background())
	if err != nil || len(parts) != 3 {
		t.Errorf("Parts = %d, %v; want the app, the font and the files", len(parts), err)
	}
}
