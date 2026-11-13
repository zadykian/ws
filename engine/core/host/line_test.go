package host_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

func TestLine(t *testing.T) {
	for _, tt := range []struct{ name, before, match, want string }{
		{"missing file", "", "", "PermitRootLogin no\n"},
		{"appended", "a\nb", "", "a\nb\nPermitRootLogin no\n"},
		{"present", "a\nPermitRootLogin no\nb\n", "", "a\nPermitRootLogin no\nb\n"},
		{"replaced", "a\n#PermitRootLogin yes\nb\n", `^#?PermitRootLogin\b`, "a\nPermitRootLogin no\nb\n"},
		{"no match", "a\n", `^#?PermitRootLogin\b`, "a\nPermitRootLogin no\n"},
	} {
		m := effecttest.NewMachine()
		if tt.before != "" {
			m.Files["/f"] = []byte(tt.before)
		}
		l := host.Line{M: m, Path: "/f", Text: "PermitRootLogin no", Match: tt.match, Mode: 0o644}
		applyAll(t, l)
		if got := string(m.Files["/f"]); got != tt.want {
			t.Errorf("%s: file = %q, want %q", tt.name, got, tt.want)
		}
		if again := applyAll(t, l); again != nil {
			t.Errorf("%s: second plan = %q, want none", tt.name, again)
		}
	}
}

func TestTwoLinesOfOneFileKeepEachOther(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/f"] = []byte("a\n")
	one := host.Line{M: m, Path: "/f", Text: "one", Mode: 0o644}
	two := host.Line{M: m, Path: "/f", Text: "two", Mode: 0o644}
	g := host.Group{Title: "F", Parts: []host.Resource{one, two}}
	if _, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/f"]); got != "a\none\ntwo\n" {
		t.Errorf("file = %q, want both lines", got)
	}
}
