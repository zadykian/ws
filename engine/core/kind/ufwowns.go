package kind

import (
	"fmt"
	"maps"
	"slices"

	"github.com/zadykian/ws/engine/core/resource/ufw"
)

// ufwFields gives a field of ufw's rule by its key in a ufw state's rule, for owns.
var ufwFields = map[string]func(ufw.Rule) string{
	"action":    func(r ufw.Rule) string { return r.Action },
	"proto":     func(r ufw.Rule) string { return r.Proto },
	"to":        func(r ufw.Rule) string { return r.To },
	"to_port":   func(r ufw.Rule) string { return r.ToPort },
	"from":      func(r ufw.Rule) string { return r.From },
	"from_port": func(r ufw.Rule) string { return r.FromPort },
	"dir":       func(r ufw.Rule) string { return r.Dir },
	"interface": func(r ufw.Rule) string { return r.Interface },
}

// owns returns which rules of the comment the state owns. With Owns, it owns a rule where each
// field that Owns names holds its value. There "any" stands for any address, as in a rule. So a
// rule made by hand with the comment, in another shape, stays. Without Owns the state owns each
// rule of the comment.
func (s ufwState) owns() (func(ufw.Rule) bool, error) {
	if s.Owns == nil {
		return nil, nil
	}
	type field struct {
		get  func(ufw.Rule) string
		want string
	}
	fields := make([]field, 0, len(s.Owns))
	for _, key := range slices.Sorted(maps.Keys(s.Owns)) {
		get, ok := ufwFields[key]
		if !ok {
			return nil, fmt.Errorf("owns: %q is no field of a rule", key)
		}
		want := s.Owns[key]
		if want == "any" && (key == "to" || key == "from") {
			want = ufw.Any
		}
		fields = append(fields, field{get, want})
	}
	return func(r ufw.Rule) bool {
		return !slices.ContainsFunc(fields, func(f field) bool { return f.get(r) != f.want })
	}, nil
}
