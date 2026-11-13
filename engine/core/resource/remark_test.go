package resource_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

func TestRemarkShowsItselfAndAppliesNothing(t *testing.T) {
	r := resource.Remark("would wait")
	if err := r.Apply(context.Background()); err != nil || r.String() != "would wait" {
		t.Errorf("Apply = %v, String = %q; want nil, would wait", err, r)
	}
	if step.KindOf(r) != step.OtherChange {
		t.Errorf("kind = %s, want other", step.KindOf(r))
	}
}
