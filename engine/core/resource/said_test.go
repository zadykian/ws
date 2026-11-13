package resource_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

type kindly struct{ change }

func (kindly) Kind() step.ChangeKind { return step.Restart }
func (kindly) Item() step.Item       { return step.Item{Name: "an item"} }

func TestSaidSaysEachHeadlineOnceApplied(t *testing.T) {
	var log []string
	out := &effecttest.Out{}
	boom := errors.New("boom")
	s := resource.Said{Out: out, Part: part{change{"one", &log, nil}, change{"two", &log, boom}, change{"three", &log, nil}}}
	changes, err := s.Plan(context.Background())
	if err != nil || len(changes) != 3 {
		t.Fatalf("Plan = %v, %v; want three changes", changes, err)
	}
	for _, c := range changes {
		_ = c.Apply(context.Background())
	}
	if len(out.Lines) != 2 || out.Lines[0] != "say: one" || out.Lines[1] != "say: three" {
		t.Errorf("said %q; want one and three, not the failed two", out.Lines)
	}
}

func TestSaidKeepsAChangesKindAndItem(t *testing.T) {
	var log []string
	changes, _ := resource.Said{Out: &effecttest.Out{}, Part: part{kindly{change{"k", &log, nil}}}}.Plan(context.Background())
	if it, ok := changes[0].(step.Itemized); !ok || it.Item().Name != "an item" || step.KindOf(changes[0]) != step.Restart {
		t.Errorf("%T: kind %s, itemized %v; want restart and an item", changes[0], step.KindOf(changes[0]), ok)
	}
}

type forcedChange struct{ change }

func (forcedChange) Forced() bool { return true }

type warnedChange struct{ change }

func (warnedChange) Warned() bool { return false }

type earlyChange struct{ change }

func (earlyChange) Early() bool { return true }

type titledChange struct{ change }

func (titledChange) Title() string { return "a title" }

type excerptedChange struct{ change }

func (excerptedChange) Excerpt() string { return "an excerpt" }

func TestSaidRefusesAChangeWithAMarkerItWouldDrop(t *testing.T) {
	var log []string
	c := change{"m", &log, nil}
	for name, marked := range map[string]resource.Change{
		"forced": forcedChange{c}, "warned": warnedChange{c}, "early": earlyChange{c},
		"titled": titledChange{c}, "excerpted": excerptedChange{c},
	} {
		changes, err := resource.Said{Out: &effecttest.Out{}, Part: part{marked}}.Plan(context.Background())
		if err == nil || changes != nil {
			t.Errorf("%s: Plan = %v, %v; want an error and no changes", name, changes, err)
		}
	}
}

func TestSaidStillSaysAnUnmarkedChange(t *testing.T) {
	var log []string
	out := &effecttest.Out{}
	changes, err := resource.Said{Out: out, Part: part{change{"plain", &log, nil}}}.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", changes, err)
	}
	_ = changes[0].Apply(context.Background())
	if len(out.Lines) != 1 || out.Lines[0] != "say: plain" {
		t.Errorf("said %q; want plain", out.Lines)
	}
}
