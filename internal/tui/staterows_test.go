package tui

import "testing"

func TestPrintable(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"words and lines stay", "a b\nc", "a b\nc"},
		{"a tab takes spaces", "a\tb", "a    b"},
		{"an escape goes", "a\x1b[2Jb", "a[2Jb"},
		{"a carriage return goes", "a\r\nb", "a\nb"},
		{"a C1 code goes", "a\u009bb", "ab"},
		{"bytes that are no text show", "a\xffb", "a�b"},
	}
	for _, c := range cases {
		if got := printable(c.in); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
