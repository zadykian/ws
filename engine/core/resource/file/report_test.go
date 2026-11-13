package file_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// reportOf returns the one item of a resource's report.
func reportOf(t *testing.T, r step.Reporter) step.Item {
	items, err := r.Report(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v, want one", items)
	}
	return items[0]
}
