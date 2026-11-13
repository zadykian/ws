package resource_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

func TestLazyMakesItsPartsAtEachPlan(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	content := "one"
	lazy := resource.Lazy{Title: "L", Parts: func(context.Context) ([]resource.Resource, error) {
		return []resource.Resource{file.File{M: m, Path: "/srv/a", Content: []byte(content), Mode: 0o644}}, nil
	}}
	if changed, err := step.Run(t.Context(), []step.Step{lazy}, out, false); err != nil || !changed {
		t.Fatalf("Run = %v, %v; want the file written", changed, err)
	}
	content = "two"
	p, err := lazy.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Errorf("Plan after the content moved = %v, %v; want one change", p.Changes, err)
	}
}

func TestLazyFailsWhereItsPartsFail(t *testing.T) {
	boom := errors.New("boom")
	lazy := resource.Lazy{Title: "L", Parts: func(context.Context) ([]resource.Resource, error) { return nil, boom }}
	if _, err := lazy.Plan(t.Context()); !errors.Is(err, boom) {
		t.Errorf("Plan = %v, want boom", err)
	}
}
