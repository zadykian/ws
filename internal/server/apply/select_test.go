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
