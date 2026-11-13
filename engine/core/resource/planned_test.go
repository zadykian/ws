package resource_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// inert is a change that does not apply itself.
type inert string

func (i inert) String() string { return string(i) }

func TestPlannedReturnsItsChangesAndReportsWhatItSaw(t *testing.T) {
	var log []string
	seen := resource.Group{Parts: []resource.Resource{}}
	p := resource.Planned{Changes: []step.Change{change{"one", &log, nil}}}
	changes, err := p.Plan(context.Background())
	if err != nil || len(changes) != 1 || changes[0].String() != "one" {
		t.Fatalf("Plan = %v, %v; want one change", changes, err)
	}
	if items, err := p.Report(context.Background()); items != nil || err != nil {
		t.Errorf("Report without Seen = %v, %v; want nothing", items, err)
	}
	p.Seen = seen
	if _, err := p.Report(context.Background()); err != nil {
		t.Errorf("Report = %v", err)
	}
}

func TestPlannedStopsAtItsErrorAndAtAChangeThatDoesNotApply(t *testing.T) {
	boom := errors.New("boom")
	if _, err := (resource.Planned{Err: boom}).Plan(context.Background()); !errors.Is(err, boom) {
		t.Errorf("Plan err = %v, want boom", err)
	}
	_, err := resource.Planned{Changes: []step.Change{inert("odd")}}.Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "odd does not apply itself") {
		t.Errorf("Plan err = %v, want the change named", err)
	}
}
