package values

import (
	"strings"
	"testing"
)

func TestSpanFindsTheLinesOfAKey(t *testing.T) {
	lines := strings.SplitAfter("---\n# a\nWS_BASE_TIMEZONE: x\n\nWS_SIGNOZ_DASHBOARDS:\n  - git\n- jq\n\n# b\nWS_USER : y\n"+
		"WS_USER_NAME: z\n", "\n")
	lines = lines[:len(lines)-1]
	for name, want := range map[string][2]int{
		"WS_BASE_TIMEZONE": {2, 3}, "WS_SIGNOZ_DASHBOARDS": {4, 7}, "WS_USER": {9, 10}, "WS_HOME": {11, 11},
	} {
		if start, end := span(lines, name); start != want[0] || end != want[1] {
			t.Errorf("%s: span = %d, %d, want %v", name, start, end, want)
		}
	}
}

func TestSetEditsOneLineAndKeepsTheRest(t *testing.T) {
	const file = "---\n# the machine\nWS_BASE_TIMEZONE: UTC\n\n# the link\nWS_SIGNOZ_DASHBOARDS:\n  - git\n  - jq\n\n# the cable\n" +
		"WS_CABLE_PORT: enp1\n"
	for _, tc := range []struct {
		name, data, key, value, want string
	}{
		{"replaces", file, "WS_BASE_TIMEZONE", "Asia/Tbilisi",
			strings.Replace(file, "UTC", "Asia/Tbilisi", 1)},
		{"replaces a block list", file, "WS_SIGNOZ_DASHBOARDS", "git, tmux",
			strings.Replace(file, "WS_SIGNOZ_DASHBOARDS:\n  - git\n  - jq\n", "WS_SIGNOZ_DASHBOARDS: [git, tmux]\n", 1)},
		{"adds at the end", file, "WS_BASE_SWAPPINESS", "20", file + "WS_BASE_SWAPPINESS: 20\n"},
		{"takes a line out", file, "WS_BASE_TIMEZONE", "", strings.Replace(file, "WS_BASE_TIMEZONE: UTC\n", "", 1)},
		{"takes a list out", file, "WS_SIGNOZ_DASHBOARDS", "", strings.Replace(file, "WS_SIGNOZ_DASHBOARDS:\n  - git\n  - jq\n", "", 1)},
		{"leaves a key that no line holds", file, "WS_BASE_SWAPPINESS", "", file},
		{"starts an empty file", "", "WS_USER", "ada", "---\nWS_USER: ada\n"},
		{"ends the last line", "WS_USER: ada", "WS_BASE_SWAPPINESS", "20", "WS_USER: ada\nWS_BASE_SWAPPINESS: 20\n"},
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
	once, err := Set([]byte("---\n"), "WS_IKEWARD_DOMAINS", "a.example,b.example")
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Set(once, "WS_IKEWARD_DOMAINS", "a.example, b.example")
	if err != nil || string(twice) != string(once) {
		t.Errorf("twice = %q, %v, want %q", twice, err, once)
	}
}

// The new line holds no home of its own: the file's home judges its path.
func TestSetTakesAKeyBelowTheFilesHome(t *testing.T) {
	for _, tc := range [][3]string{
		{"WS_USER: root\n", "WS_GIT_SIGNING_PUB_KEY", "/root/.ssh/k.pub"},
		{"WS_HOME: /srv/x\n", "WS_GIT_SIGNING_PUB_KEY", "/srv/x/k.pub"},
	} {
		if _, err := Set([]byte(tc[0]), tc[1], tc[2]); err != nil {
			t.Errorf("%s at %s in %q: %v", tc[1], tc[2], tc[0], err)
		}
	}
	if _, err := Set([]byte("WS_USER: root\n"), "WS_GIT_SIGNING_PUB_KEY", "/home/zadykian/k.pub"); err == nil ||
		!strings.Contains(err.Error(), "is outside WS_HOME /root") {
		t.Errorf("a key outside the file's home: %v", err)
	}
}

func TestSetRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, data, key, value, want string
	}{
		{"an unknown key", "", "no_such_key", "1", "no line holds no_such_key"},
		{"a table", "", "WS_FANS_CURVES", "1", "no line holds WS_FANS_CURVES"},
		{"a wrong kind", "", "WS_BASE_SWAPPINESS", "ten", "a whole number"},
		{"a broken file", "WS_USER: [\n", "WS_HOME", "/h", "yaml"},
		{"a key of a quoted name", "\"WS_USER\": ada\n", "WS_USER", "bob", "edit it by hand"},
		{"a document that ended", "WS_HOME: /h\n...\n", "WS_USER", "bob", "edit it by hand"},
	} {
		if _, err := Set([]byte(tc.data), tc.key, tc.value); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}
