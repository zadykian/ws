package host_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
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

func TestFile(t *testing.T) {
	m := effecttest.NewMachine()
	f := host.File{M: m, Path: "/srv/a", Content: []byte("new\n"), Mode: 0o600, UID: 5, GID: 6}
	if got := applyAll(t, f); len(got) != 1 || !strings.HasPrefix(got[0], "create /srv/a") {
		t.Errorf("first plan = %q, want a create", got)
	}
	if got := applyAll(t, f); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}
