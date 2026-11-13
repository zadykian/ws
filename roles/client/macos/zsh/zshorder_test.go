package zsh_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/client/macos/zsh"
)

const linkWarn = "warn: /h/.config is a link to /dots/config: ws's files go there"

// linkedConfig is a Mac whose ~/.config is a link and whose ws directory has the wrong mode, so
// that a plan writes under the link.
func linkedConfig() (*effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/dots/config"], m.Dirs["/h/.config/ws"] = 0o755, 0o755
	m.Links["/h/.config"] = "/dots/config"
	return m, out
}

// A warning comes in this order: the hook's, or the line's, then the link's. A plan that stops on
// ~/.zshrc says the link's warning before its error.
func TestZshWarnsInOrderUnderALinkedConfig(t *testing.T) {
	notWs, _ := linkedConfig()
	notWs.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	linkedRc, _ := linkedConfig()
	linkedRc.Files["/h/.config/ws/zshrc"] = zsh.Hook
	linkedRc.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	linkedRc.Links["/h/.zshrc"] = "/dots/zshrc"
	unreadable, _ := linkedConfig()
	unreadable.Files["/h/.config/ws/zshrc"] = zsh.Hook
	unreadable.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	unreadable.Root["/h/.zshrc"] = effect.Attrs{}
	noHook, _ := linkedConfig()
	noHook.Dirs["/h/.config/ws"] = 0o700
	noHook.Files["/h/.zshrc"] = []byte("source ~/.config/ws/zshrc\n")
	noLine, _ := linkedConfig()
	noLine.Dirs["/h/.config/ws"] = 0o700
	noLine.Files["/h/.config/ws/zshrc"] = zsh.Hook
	noLine.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	cases := []struct {
		name    string
		m       *effecttest.Machine
		changes int
		warns   []string
		fails   bool
	}{
		{"hook not ws's", notWs, 1, []string{
			"warn: /h/.config/ws/zshrc is not ws's: move it away, and the next run writes ws's and sources it", linkWarn}, false},
		{"linked zshrc without the line", linkedRc, 1, []string{
			"warn: /h/.zshrc is a link to /dots/zshrc: add 'source ~/.config/ws/zshrc' at the end of that file", linkWarn}, false},
		{"only the hook missing", noHook, 1, []string{linkWarn}, false},
		{"only the line missing", noLine, 2, []string{linkWarn}, false},
		{"zshrc unreadable", unreadable, 0, []string{linkWarn}, true},
	}
	for _, c := range cases {
		out := &effecttest.Out{}
		p, err := zshStep(c.m, out).Plan(context.Background())
		if (err != nil) != c.fails || len(p.Changes) != c.changes || !slices.Equal(out.Lines, c.warns) {
			t.Errorf("%s: Plan = %d changes, %v, said %q; want %d, fails %v, %q",
				c.name, len(p.Changes), err, out.Lines, c.changes, c.fails, c.warns)
		}
	}
}

// The warnings come from the last part alone: the parts before it plan the changes and say nothing.
func TestZshWarnsFromItsLastPartAlone(t *testing.T) {
	m, out := linkedConfig()
	m.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	parts, err := zshStep(m, out).Parts(context.Background())
	if err != nil || len(parts) != 4 {
		t.Fatalf("Parts = %d, %v; want 4", len(parts), err)
	}
	if len(out.Lines) != 0 {
		t.Fatalf("Parts said %q; want nothing", out.Lines)
	}
	if _, err := (resource.Group{Parts: parts[:3]}).Plan(context.Background()); err != nil || len(out.Lines) != 0 {
		t.Fatalf("the first three parts: %v, said %q; want no warning", err, out.Lines)
	}
	if _, err := parts[3].Plan(context.Background()); err != nil || len(out.Lines) != 2 {
		t.Errorf("the last part: %v, said %q; want the two warnings", err, out.Lines)
	}
}

// A plan says each warning once, and a report after it, as --check and status run it, says none.
func TestZshWarningsDoNotDoubleUnderCheckOrShowInStatus(t *testing.T) {
	m, out := linkedConfig()
	m.Files["/h/.config/ws/zshrc"] = []byte("echo mine\n")
	s := zshStep(m, out)
	if _, err := s.Plan(context.Background()); err != nil || len(out.Lines) != 2 {
		t.Fatalf("Plan = %v, said %q; want the two warnings", err, out.Lines)
	}
	if _, err := s.Report(context.Background()); err != nil || len(out.Lines) != 2 {
		t.Errorf("Report = %v, said %q; want no more", err, out.Lines)
	}
}
