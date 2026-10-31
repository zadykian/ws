package conf

import "testing"

func TestErrorIn(t *testing.T) {
	if got := (Error{2, "x"}).In("f"); got != "f, line 2: x" {
		t.Errorf("got %q", got)
	}
	if got := (Error{0, "x"}).In("f"); got != "f: x" {
		t.Errorf("got %q", got)
	}
}
