package account_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
)

func applyAll(t *testing.T, r resource.Resource) []string {
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
