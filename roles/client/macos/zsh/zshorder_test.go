package zsh_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

const linkedWarn = "warn: /h/.zshrc is a link to /dots/zshrc: add 'source /etc/ws/zshrc' at the end of that file"

func linkedRc() (*effecttest.Machine, *effecttest.Out) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files["/h/.zshrc"] = []byte("export EDITOR=vi\n")
	m.Links["/h/.zshrc"] = "/dots/zshrc"
	return m, out
}

// A plan that stops on ~/.zshrc says the link's warning, and a link that lacks the line stops
// nothing.
func TestZshWarnsOfTheLinkedZshrcBeforeItsError(t *testing.T) {
	linked, out := linkedRc()
	p, err := zshStep(linked, out).Plan(context.Background())
	if err != nil || len(p.Changes) != 2 || !slices.Equal(out.Lines, []string{linkedWarn}) {
		t.Errorf("linked: Plan = %d changes, %v, said %q; want the hook's two and the warning", len(p.Changes), err, out.Lines)
	}
	unreadable, out := linkedRc()
	unreadable.Root["/h/.zshrc"] = effect.Attrs{}
	p, err = zshStep(unreadable, out).Plan(context.Background())
	if err == nil || len(p.Changes) != 0 {
		t.Errorf("unreadable: Plan = %d changes, %v; want its error", len(p.Changes), err)
	}
}

// The warning comes from the last part alone: the parts before it plan the changes and say nothing.
func TestZshWarnsFromItsLastPartAlone(t *testing.T) {
	m, out := linkedRc()
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
	if _, err := parts[3].Plan(context.Background()); err != nil || len(out.Lines) != 1 {
		t.Errorf("the last part: %v, said %q; want the warning", err, out.Lines)
	}
}

// A plan says its warning once, and a report after it, as --check and status run it, says none.
func TestZshWarningDoesNotDoubleUnderCheckOrShowInStatus(t *testing.T) {
	m, out := linkedRc()
	s := zshStep(m, out)
	if _, err := s.Plan(context.Background()); err != nil || len(out.Lines) != 1 {
		t.Fatalf("Plan = %v, said %q; want the warning", err, out.Lines)
	}
	if _, err := s.Report(context.Background()); err != nil || len(out.Lines) != 1 {
		t.Errorf("Report = %v, said %q; want no more", err, out.Lines)
	}
}
