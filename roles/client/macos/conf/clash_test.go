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
		"same domain":   {"WS_RESOLVERS=corp.test=10.0.0.1", "WS_RESOLVERS=\"a.test=10.0.0.2 corp.test=10.0.0.3\"", "WS_RESOLVERS's corp.test is server lab's too: /etc/resolver holds one file a domain"},
	} {
		got := clashes(parse(t, tc.own), sibling(tc.theirs))
		if len(got) != 1 || !strings.Contains(got[0].Msg, tc.want) {
			t.Errorf("%s: clashes = %+v, want %q", name, got, tc.want)
		}
	}
}

func TestClashesAllowTheServersOfTheThreeTargets(t *testing.T) {
	lan := "WS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=ws.example.com\nWS_REMOTE_PORT=22022\nWS_TUNNEL_NETWORKS=10.20.0.0/16\nWS_RESOLVERS=a.test=10.20.0.53\n"
	cloud := "WS_TARGET=cloud\nWS_REMOTE_PORT=22\nWS_REMOTE=198.51.100.7\nWS_TUNNEL_NETWORKS=10.30.0.0/16\nWS_RESOLVERS=b.test=10.30.0.53\n"
	vm := "WS_TARGET=mac-vm\nWS_LAN=192.168.64.0/24\nWS_LAN_ADDRESS=192.168.64.5\nWS_TUNNEL_NETWORKS=10.40.0.0/16\n"
	keys := []string{lan, cloud, vm}
	for i, a := range keys {
		for j, b := range keys {
			if i != j {
				if got := clashes(parse(t, a), sibling(b)); len(got) != 0 {
					t.Errorf("servers %d and %d: clashes = %+v, want none", i, j, got)
				}
			}
		}
	}
}

func TestClashesIgnoreAServerWithNoKeys(t *testing.T) {
	if got := clashes(parse(t, "WS_TUNNEL_NETWORKS=10.1.0.0/24\nWS_RESOLVERS=a.test=10.0.0.1"), sibling("")); len(got) != 0 {
		t.Errorf("clashes = %+v, want none", got)
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
