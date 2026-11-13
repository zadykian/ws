package zed_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// freshZed is a Mac with neither the app nor the font, where Homebrew has neither cask.
func freshZed() (*effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	for _, name := range []string{"zed", "font-jetbrains-mono-nerd-font"} {
		m.Results["brew list --cask "+name] = effecttest.Result{Err: errors.New("no")}
		m.Results["brew install --cask "+name] = effecttest.Result{}
	}
	return m, out
}

// Apply and --check say the casks first, then the files, in this order.
func TestZedPlansAndAppliesInOrder(t *testing.T) {
	m, out := freshZed()
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
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
		"would run brew install --cask zed",
		"would run brew install --cask font-jetbrains-mono-nerd-font",
		"would make /h/.config, mode 755",
		"would make /h/.config/zed, mode 755",
		"would create /h/.config/zed/global_settings.json",
		"would create /h/.config/zed/keymap.json",
		"would create /h/.config/zed/settings.json",
	}
	if !slices.Equal(heads, want) {
		t.Fatalf("headlines %q; want %q", heads, want)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	said := []string{"say: installing zed", "say: installing font-jetbrains-mono-nerd-font",
		"say: wrote /h/.config/zed/global_settings.json", "say: wrote /h/.config/zed/keymap.json",
		"say: wrote /h/.config/zed/settings.json"}
	if !slices.Equal(out.Lines, said) {
		t.Errorf("said %q; want %q", out.Lines, said)
	}
}

// A report lists the app, the font, the directories, the two files, then Zed's own.
func TestZedReportsInOrder(t *testing.T) {
	m, out := freshZed()
	items, err := zedStep(m, out).Report(context.Background())
	var names []string
	for _, it := range items {
		names = append(names, it.Name)
	}
	want := []string{"zed", "font-jetbrains-mono-nerd-font", "/h/.config", zedConfig[:len(zedConfig)-1],
		zedConfig + "global_settings.json", zedConfig + "keymap.json", zedConfig + "settings.json"}
	if err != nil || !slices.Equal(names, want) {
		t.Errorf("items %q, %v; want %q", names, err, want)
	}
}

// The step is the app's cask, the font's, then the files, which the app's install holds back.
func TestZedPartsAreTheAppTheFontAndTheFiles(t *testing.T) {
	m, out := freshZed()
	parts, err := zedStep(m, out).Parts(context.Background())
	if err != nil || len(parts) != 3 {
		t.Errorf("Parts = %d, %v; want the app, the font and the files", len(parts), err)
	}
}
