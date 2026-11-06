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

func TestSetEditsOneLineAndKeepsTheRest(t *testing.T) {
	const file = "---\n# the machine\nbase_timezone: UTC\n\n# the link\nbase_packages:\n  - git\n  - jq\n\n# the cable\n" +
		"network_direct_link_interface: enp1\n"
	for _, tc := range []struct {
		name, data, key, value, want string
	}{
		{"replaces", file, "base_timezone", "Asia/Tbilisi",
			strings.Replace(file, "UTC", "Asia/Tbilisi", 1)},
		{"replaces a block list", file, "base_packages", "git, tmux",
			strings.Replace(file, "base_packages:\n  - git\n  - jq\n", "base_packages: [git, tmux]\n", 1)},
		{"adds at the end", file, "base_swappiness", "20", file + "base_swappiness: 20\n"},
		{"takes a line out", file, "base_timezone", "", strings.Replace(file, "base_timezone: UTC\n", "", 1)},
		{"takes a list out", file, "base_packages", "", strings.Replace(file, "base_packages:\n  - git\n  - jq\n", "", 1)},
		{"leaves a key that no line holds", file, "base_swappiness", "", file},
		{"starts an empty file", "", "ws_user", "ada", "---\nws_user: ada\n"},
		{"ends the last line", "ws_user: ada", "base_swappiness", "20", "ws_user: ada\nbase_swappiness: 20\n"},
	} {
		got, err := Set([]byte(tc.data), tc.key, tc.value)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
		} else if string(got) != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}

func TestSetTwiceChangesNothing(t *testing.T) {
	once, err := Set([]byte("---\n"), "ikeward_domains", "a.example,b.example")
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Set(once, "ikeward_domains", "a.example, b.example")
	if err != nil || string(twice) != string(once) {
		t.Errorf("twice = %q, %v, want %q", twice, err, once)
	}
}

func TestSetRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, data, key, value, want string
	}{
		{"an unknown key", "", "no_such_key", "1", "no line holds no_such_key"},
		{"a table", "", "fans_curves", "1", "no line holds fans_curves"},
		{"a wrong kind", "", "base_swappiness", "ten", "a whole number"},
		{"a broken file", "ws_user: [\n", "ws_home", "/h", "yaml"},
		{"a key of a quoted name", "\"ws_user\": ada\n", "ws_user", "bob", "edit it by hand"},
		{"a document that ended", "ws_home: /h\n...\n", "ws_user", "bob", "edit it by hand"},
	} {
		if _, err := Set([]byte(tc.data), tc.key, tc.value); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}
