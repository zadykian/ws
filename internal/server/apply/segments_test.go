package apply

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

func steps(values.Values, effect.Machine, effect.Out) []step.Step { return nil }

func TestSegments(t *testing.T) {
	list := []roles.Role{{Name: "a"}, {Name: "b"}, {Name: "c", Steps: steps}, {Name: "d"}, {Name: "e", Steps: steps}}
	for _, tc := range []struct {
		name string
		only []string
		want []string // a playbook segment as "playbook a,b", a ported one as its name
	}{
		{"all", nil, []string{"playbook a,b", "c", "playbook d", "e"}},
		{"in the list's order", []string{"e", "b", "a"}, []string{"playbook a,b", "e"}},
		{"a ported role between", []string{"a", "d"}, []string{"playbook a,d"}},
		{"ported alone", []string{"c"}, []string{"c"}},
	} {
		segs, err := Segments(list, tc.only)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, s := range segs {
			if s.Playbook != nil {
				got = append(got, "playbook "+strings.Join(s.Playbook, ","))
			} else {
				got = append(got, s.Role.Name)
			}
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: segments = %q, want %q", tc.name, got, tc.want)
		}
	}
	if _, err := Segments(list, []string{"x"}); err == nil || !strings.Contains(err.Error(), `no role "x"; the roles are a, b, c, d, e`) {
		t.Errorf("unknown role: err = %v", err)
	}
}
