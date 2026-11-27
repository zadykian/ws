package resource_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// inert is a change that does not apply itself.
type inert string

func (i inert) String() string { return string(i) }

func TestPlannedReturnsItsChanges(t *testing.T) {
	var log []string
	p := resource.Planned{Changes: []step.Change{change{"one", &log, nil}}}
	changes, err := p.Plan(context.Background())
	if err != nil || len(changes) != 1 || changes[0].String() != "one" {
		t.Fatalf("Plan = %v, %v; want one change", changes, err)
	}
}

func TestPlannedStopsAtAChangeThatDoesNotApply(t *testing.T) {
	_, err := resource.Planned{Changes: []step.Change{inert("odd")}}.Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "odd does not apply itself") {
		t.Errorf("Plan err = %v, want the change named", err)
	}
}
