// Package resourcetest holds what the tests of the resource packages share: applying a resource
// and stating a pin's sum. A failure names the call that made it, as no test function hides a
// frame.
package resourcetest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
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

// SumOf returns the hex SHA-256 of body, as a pin states it.
func SumOf(body string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(body))) }
