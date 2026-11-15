package conf

import (
	"strings"
	"testing"
)

// sibling returns a server lab whose file holds the keys. Lab sits at slot 15 where they name none
// and are not empty, so that a row clashes on the keys it names alone.
func sibling(keys string) []Sibling {
	if keys != "" && !strings.Contains(keys, "WS_SLOT") {
		keys = "WS_SLOT=15\n" + keys
	}
	c, _ := ServerFile.Parse(keys)
	return []Sibling{{Name: "lab", File: ServerPath(macPath, "lab"), Conf: c}}
}

func TestClashesRefuseWhatTwoServersCannotShare(t *testing.T) {
	for name, tc := range map[string]struct{ own, theirs, want string }{
		"own inside":           {"WS_TUNNEL_NETWORKS=10.1.0.0/24", "WS_TUNNEL_NETWORKS=10.1.0.0/16", "WS_TUNNEL_NETWORKS's 10.1.0.0/24 overlaps server lab's 10.1.0.0/16: a network goes through one server's tunnel"},
		"theirs inside":        {"WS_TUNNEL_NETWORKS=10.1.0.0/16", "WS_TUNNEL_NETWORKS=10.1.2.0/24", "WS_TUNNEL_NETWORKS's 10.1.0.0/16 overlaps server lab's 10.1.2.0/24: a network goes through one server's tunnel"},
		"same domain":          {"WS_RESOLVERS=corp.test=10.0.0.1", "WS_RESOLVERS=\"a.test=10.0.0.2 corp.test=10.0.0.3\"", "WS_RESOLVERS's corp.test is server lab's too: /etc/resolver holds one file a domain"},
		"same slot":            {"WS_SLOT=1", "WS_SLOT=1", "WS_SLOT=1 is server lab's slot too: each server takes its own"},
		"no line is 0":         {"WS_TARGET=lan", "WS_SLOT=0", "WS_SLOT=0 is server lab's slot too: each server takes its own"},
		"same port":            {"WS_REMOTE=r.example\nWS_REMOTE_PORT=22023\nWS_SLOT=1", "WS_REMOTE=r.example\nWS_REMOTE_PORT=22022", "WS_PORT=51820 is server lab's too, behind WS_REMOTE r.example: the router forwards a port to one server"},
		"same in another case": {"WS_REMOTE=R.Example\nWS_REMOTE_PORT=22023\nWS_SLOT=1", "WS_REMOTE=r.example\nWS_REMOTE_PORT=22022", "WS_PORT=51820 is server lab's too, behind WS_REMOTE R.Example: the router forwards a port to one server"},
		"same forward":         {"WS_REMOTE=r.example\nWS_PORT=51821\nWS_SLOT=1", "WS_REMOTE=r.example\nWS_PORT=51820", "WS_REMOTE_PORT=22022 is server lab's too, behind WS_REMOTE r.example: the router forwards a port to one server"},
		"second cable":         {"WS_CABLE_SERVICE=\"USB LAN\"", "WS_CABLE_SERVICE=\"Thunderbolt Ethernet\"", "WS_CABLE_SERVICE names a cable, as server lab's does: the cable reaches one server"},
	} {
		got := clashes(parse(t, tc.own), sibling(tc.theirs))
		if len(got) != 1 || !strings.Contains(got[0].Msg, tc.want) {
			t.Errorf("%s: clashes = %+v, want %q", name, got, tc.want)
		}
	}
}

func TestClashesAllowTheServersOfEachTargetAndOfTwoServersOfOne(t *testing.T) {
	lan := "WS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=ws.example.com\nWS_REMOTE_PORT=22022\nWS_TUNNEL_NETWORKS=10.20.0.0/16\nWS_RESOLVERS=a.test=10.20.0.53\n"
	cloud := "WS_SLOT=1\nWS_TARGET=cloud\nWS_REMOTE_PORT=22\nWS_REMOTE=198.51.100.7\nWS_TUNNEL_NETWORKS=10.30.0.0/16\nWS_RESOLVERS=b.test=10.30.0.53\n"
	vm := "WS_SLOT=2\nWS_TARGET=mac-vm\nWS_LAN=192.168.64.0/24\nWS_LAN_ADDRESS=192.168.64.5\nWS_TUNNEL_NETWORKS=10.40.0.0/16\n"
	// Each second server holds its placed slot and ports: two machines behind one router, two cloud
	// VMs, and two VMs on the Mac's one network.
	lan2 := "WS_SLOT=3\nWS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.11\nWS_REMOTE=ws.example.com\nWS_PORT=51823\nWS_REMOTE_PORT=22025\nWS_TUNNEL_NETWORKS=10.21.0.0/16\nWS_RESOLVERS=c.test=10.21.0.53\n"
	cloud2 := "WS_SLOT=4\nWS_TARGET=cloud\nWS_REMOTE_PORT=22\nWS_REMOTE=198.51.100.8\nWS_TUNNEL_NETWORKS=10.31.0.0/16\nWS_RESOLVERS=d.test=10.31.0.53\n"
	vm2 := "WS_SLOT=5\nWS_TARGET=mac-vm\nWS_LAN=192.168.64.0/24\nWS_LAN_ADDRESS=192.168.64.6\nWS_TUNNEL_NETWORKS=10.41.0.0/16\n"
	for _, pair := range [][2]string{{lan, cloud}, {lan, vm}, {cloud, vm}, {lan, lan2}, {cloud, cloud2}, {vm, vm2}} {
		for _, order := range [][2]string{pair, {pair[1], pair[0]}} {
			if got := clashes(parse(t, order[0]), sibling(order[1])); len(got) != 0 {
				t.Errorf("%q beside %q: clashes = %+v, want none", order[0], order[1], got)
			}
		}
	}
}

func TestClashesIgnoreAServerWithNoKeys(t *testing.T) {
	if got := clashes(parse(t, "WS_SLOT=1\nWS_TUNNEL_NETWORKS=10.1.0.0/24\nWS_RESOLVERS=a.test=10.0.0.1"), sibling("")); len(got) != 0 {
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
