package systemd_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

func applyAll(t *testing.T, r host.Resource) []string {
	changes, err := r.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var shown []string
	for _, c := range changes {
		shown = append(shown, c.String())
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return shown
}

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
