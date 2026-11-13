package apt_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/zadykian/ws/engine/core/host"
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

func sumOf(body string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(body))) }
