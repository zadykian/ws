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
