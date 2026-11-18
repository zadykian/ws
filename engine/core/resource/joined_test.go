package resource_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

type failing struct{}

func (failing) Plan(context.Context) ([]resource.Change, error) { return nil, errors.New("boom") }

func TestJoinedPlansThePartsInOrder(t *testing.T) {
	var log []string
	j := resource.Joined{part{change{"a", &log, nil}}, part{}, part{change{"b", &log, nil}, change{"c", &log, nil}}}
	cs, err := j.Plan(t.Context())
	var got []string
	for _, c := range cs {
		got = append(got, step.Headline(c))
	}
	if err != nil || !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("plan = %q, %v; want a, b, c", got, err)
	}
	if cs, err := append(j, failing{}).Plan(t.Context()); err == nil || cs != nil {
		t.Errorf("a failing part: plan = %q, %v; want none and its error", cs, err)
	}
}
