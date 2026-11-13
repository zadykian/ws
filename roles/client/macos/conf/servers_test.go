package conf

import "testing"

func TestIsName(t *testing.T) {
	for s, want := range map[string]bool{
		"ws": true, "a": true, "box-2": true, "abcdefghijkl": true,
		"": false, "Lab": false, "1box": false, "-box": false, "box_2": false, "abcdefghijklm": false,
	} {
		if got := IsName(s); got != want {
			t.Errorf("IsName(%q) = %v, want %v", s, got, want)
		}
	}
}
