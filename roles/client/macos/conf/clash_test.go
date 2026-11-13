package conf

import (
	"strings"
	"testing"
)

// sibling returns a server lab whose file holds the keys.
func sibling(keys string) []Sibling {
	c, _ := ServerFile.Parse(keys)
	return []Sibling{{Name: "lab", File: ServerPath(macPath, "lab"), Conf: c}}
}

func TestClashesRefuseWhatTwoServersCannotShare(t *testing.T) {
	for name, tc := range map[string]struct{ own, theirs, want string }{
		"own inside":    {"WS_TUNNEL_NETWORKS=10.1.0.0/24", "WS_TUNNEL_NETWORKS=10.1.0.0/16", "WS_TUNNEL_NETWORKS's 10.1.0.0/24 overlaps server lab's 10.1.0.0/16: a network goes through one server's tunnel"},
		"theirs inside": {"WS_TUNNEL_NETWORKS=10.1.0.0/16", "WS_TUNNEL_NETWORKS=10.1.2.0/24", "WS_TUNNEL_NETWORKS's 10.1.0.0/16 overlaps server lab's 10.1.2.0/24: a network goes through one server's tunnel"},
	} {
		got := clashes(parse(t, tc.own), sibling(tc.theirs))
		if len(got) != 1 || !strings.Contains(got[0].Msg, tc.want) {
			t.Errorf("%s: clashes = %+v, want %q", name, got, tc.want)
		}
	}
}

// parse returns the server's file's keys, which the test means to hold no refusal.
func parse(t *testing.T, keys string) Conf {
	c, errs := ServerFile.Parse(keys + "\n")
	if len(errs) > 0 {
		t.Fatalf("Parse(%q) = %+v", keys, errs)
	}
	return c
}
