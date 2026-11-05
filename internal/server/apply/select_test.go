package apply

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/roles"
)

func TestSelect(t *testing.T) {
	list := []roles.Role{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	for _, tc := range []struct {
		name string
		only []string
		want []string
	}{
		{"all", nil, []string{"a", "b", "c"}},
		{"in the list's order", []string{"c", "a"}, []string{"a", "c"}},
		{"one", []string{"b"}, []string{"b"}},
	} {
		got, err := Select(list, tc.only)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, r := range got {
			names = append(names, r.Name)
		}
		if !slices.Equal(names, tc.want) {
			t.Errorf("%s: roles = %q, want %q", tc.name, names, tc.want)
		}
	}
	if _, err := Select(list, []string{"x"}); err == nil || !strings.Contains(err.Error(), `no role "x"; the roles are a, b, c`) {
		t.Errorf("unknown role: err = %v", err)
	}
}

func TestLeftOut(t *testing.T) {
	var all Options
	if all.leftOut() != nil {
		t.Error("no Only has a filter, want none")
	}
	some := Options{Only: []string{"a/x", "a/y", "b", "b/x"}}
	left := some.leftOut()
	for _, tc := range []struct {
		role, step string
		want       bool
	}{
		{"a", "x", false},
		{"a", "y", false},
		{"a", "z", true},
		{"b", "z", false},
	} {
		if got := left(tc.role, tc.step); got != tc.want {
			t.Errorf("%s/%s: left out = %v, want %v", tc.role, tc.step, got, tc.want)
		}
	}
}
