package values

import (
	"strings"
	"testing"
)

func TestSpanFindsTheLinesOfAKey(t *testing.T) {
	lines := strings.SplitAfter("---\n# a\nbase_timezone: x\n\nbase_packages:\n  - git\n- jq\n\n# b\nws_user : y\n"+
		"ws_user_name: z\n", "\n")
	lines = lines[:len(lines)-1]
	for name, want := range map[string][2]int{
		"base_timezone": {2, 3}, "base_packages": {4, 7}, "ws_user": {9, 10}, "ws_home": {11, 11},
	} {
		if start, end := span(lines, name); start != want[0] || end != want[1] {
			t.Errorf("%s: span = %d, %d, want %v", name, start, end, want)
		}
	}
}
