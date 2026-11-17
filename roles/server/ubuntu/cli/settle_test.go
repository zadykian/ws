package cli

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

// noted is a change that notes its apply in a shared list.
type noted struct {
	name string
	done *[]string
}

func (n noted) String() string { return n.name }

func (n noted) Apply(context.Context) error {
	*n.done = append(*n.done, n.name)
	return nil
}

// fixedPlan is a resource whose plan is its changes.
type fixedPlan []resource.Change

func (p fixedPlan) Plan(context.Context) ([]resource.Change, error) { return p, nil }

func TestSettleCheckFailsOnAChange(t *testing.T) {
	var done []string
	rs := []resource.Resource{fixedPlan{noted{"a", &done}}, fixedPlan(nil), fixedPlan{noted{"b", &done}}}
	err := settle(t.Context(), &effecttest.Out{}, rs, true)
	if !errors.Is(err, errDue) || err.Error() != "to do: a; b" || done != nil {
		t.Errorf("settle = %v, applied %q; want both due, and none applied", err, done)
	}
	if err := settle(t.Context(), &effecttest.Out{}, []resource.Resource{fixedPlan(nil)}, true); err != nil {
		t.Errorf("nothing due: settle = %v", err)
	}
}

func TestSettleAppliesInOrder(t *testing.T) {
	var done []string
	out := &effecttest.Out{}
	rs := []resource.Resource{fixedPlan{noted{"a", &done}, noted{"b", &done}}, fixedPlan{noted{"c", &done}}}
	if err := settle(t.Context(), out, rs, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(done, []string{"a", "b", "c"}) || !slices.Equal(out.Lines, []string{"say: a", "say: b", "say: c"}) {
		t.Errorf("applied %q, said %q", done, out.Lines)
	}
}
