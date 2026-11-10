package main

import (
	"slices"
	"testing"
)

func TestFindings(t *testing.T) {
	m := newMirror()
	m.file = "a.go"
	m.add("first", origin{4, 3})
	m.gap()
	m.add("second", origin{9, 5})
	mirrors := map[string]*mirror{"/m/a.go.md": m}
	found, err := findings([]byte(`{"/m/a.go.md": [{"Check": "B", "Message": "late.", "Line": 4, "Span": [2, 4]},
		{"Check": "A", "Message": "early.", "Line": 2, "Span": [1, 5]}]}`), mirrors)
	want := []finding{{"a.go", "A", "early.", 4, 3}, {"a.go", "B", "late.", 9, 6}}
	if err != nil || !slices.Equal(found, want) {
		t.Errorf("got %v, %v, want %v", found, err, want)
	}
	for _, out := range []string{`{"/m/a.go.md": [{"Line": 3}]}`, `{"Code": "E100"}`} {
		if _, err := findings([]byte(out), mirrors); err == nil {
			t.Errorf("%s maps", out)
		}
	}
}
