package resource_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

type planned struct {
	changes []resource.Change
	err     error
}

func (p planned) Plan(context.Context) ([]resource.Change, error) { return p.changes, p.err }

func TestAppendAddsTheChangesAfterThoseThereAlready(t *testing.T) {
	cs := []step.Change{resource.Remark("a")}
	got, err := resource.Append(context.Background(), cs, planned{changes: []resource.Change{resource.Remark("b")}})
	if err != nil || len(got) != 2 || got[1].String() != "b" {
		t.Errorf("Append = %v, %v; want a, then b", got, err)
	}
}

func TestAppendKeepsTheChangesOfAFailedPlan(t *testing.T) {
	boom := errors.New("boom")
	got, err := resource.Append(context.Background(), nil, planned{changes: []resource.Change{resource.Remark("b")}, err: boom})
	if !errors.Is(err, boom) || len(got) != 1 {
		t.Errorf("Append = %v, %v; want b and the error", got, err)
	}
}
