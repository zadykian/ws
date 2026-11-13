// Package resourcetest holds what the tests of the resource packages share: applying a resource
// and reading its report. A failure names the call that made it, as no test function hides a
// frame.
package resourcetest

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// ApplyAll plans r, applies each change in order and returns what each change shows. It fails the
// test where the plan or an apply fails.
func ApplyAll(t testing.TB, r resource.Resource) []string {
	changes, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("resourcetest.ApplyAll: plan: %v", err)
	}
	var shown []string
	for _, c := range changes {
		shown = append(shown, c.String())
		if err := c.Apply(context.Background()); err != nil {
			t.Fatalf("resourcetest.ApplyAll: apply %q: %v", c, err)
		}
	}
	return shown
}

// ReportOf returns the one item of a resource's report. It fails the test where the report fails
// or holds another count of items.
func ReportOf(t testing.TB, r step.Reporter) step.Item {
	items, err := r.Report(context.Background())
	if err != nil {
		t.Fatalf("resourcetest.ReportOf: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("resourcetest.ReportOf: items = %+v, want one", items)
	}
	return items[0]
}
