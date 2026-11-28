package values

import (
	"strings"
	"testing"
)

// sibling returns a server lab whose file holds the keys. Lab sits at slot 15 where they name none
// and are not empty, so that a row clashes on the keys it names alone.
func sibling(keys string) []Sibling {
	if keys != "" && !strings.Contains(keys, "WS_SLOT") {
		keys = Text("WS_SLOT", "15") + keys
	}
	c, _ := ServerFile.Parse(keys)
	return []Sibling{{Name: "lab", File: ServerPath(macPath, "lab"), Conf: c}}
}

func TestClashesRefuseWhatTwoServersCannotShare(t *testing.T) {
	for name, tc := range map[string]struct{ own, theirs, want string }{
		"own inside":           {Text("WS_TUNNEL_NETWORKS", "10.1.0.0/24"), Text("WS_TUNNEL_NETWORKS", "10.1.0.0/16"), "WS_TUNNEL_NETWORKS's 10.1.0.0/24 overlaps server lab's 10.1.0.0/16: a network goes through one server's tunnel"},
		"theirs inside":        {Text("WS_TUNNEL_NETWORKS", "10.1.0.0/16"), Text("WS_TUNNEL_NETWORKS", "10.1.2.0/24"), "WS_TUNNEL_NETWORKS's 10.1.0.0/16 overlaps server lab's 10.1.2.0/24: a network goes through one server's tunnel"},
		"same domain":          {Text("WS_RESOLVERS", "corp.test=10.0.0.1"), Text("WS_RESOLVERS", "a.test=10.0.0.2 corp.test=10.0.0.3"), "WS_RESOLVERS's corp.test is server lab's too: /etc/resolver holds one file a domain"},
		"same slot":            {Text("WS_SLOT", "1"), Text("WS_SLOT", "1"), "WS_SLOT=1 is server lab's slot too: each server takes its own"},
		"no line is 0":         {Text("WS_TARGET", "lan"), Text("WS_SLOT", "0"), "WS_SLOT=0 is server lab's slot too: each server takes its own"},
		"same port":            {Text("WS_REMOTE", "r.example", "WS_REMOTE_PORT", "22023", "WS_SLOT", "1"), Text("WS_REMOTE", "r.example", "WS_REMOTE_PORT", "22022"), "WS_TUNNEL_PORT=51820 is server lab's too, behind WS_REMOTE r.example: the router forwards a port to one server"},
		"same in another case": {Text("WS_REMOTE", "R.Example", "WS_REMOTE_PORT", "22023", "WS_SLOT", "1"), Text("WS_REMOTE", "r.example", "WS_REMOTE_PORT", "22022"), "WS_TUNNEL_PORT=51820 is server lab's too, behind WS_REMOTE R.Example: the router forwards a port to one server"},
		"same forward":         {Text("WS_REMOTE", "r.example", "WS_TUNNEL_PORT", "51821", "WS_SLOT", "1"), Text("WS_REMOTE", "r.example", "WS_TUNNEL_PORT", "51820"), "WS_REMOTE_PORT=22022 is server lab's too, behind WS_REMOTE r.example: the router forwards a port to one server"},
		"second cable":         {Text("WS_CABLE_SERVICE", "USB LAN"), Text("WS_CABLE_SERVICE", "Thunderbolt Ethernet"), "WS_CABLE_SERVICE names a cable, as server lab's does: the cable reaches one server"},
	} {
		got := clashes(parse(t, tc.own), sibling(tc.theirs))
		if len(got) != 1 || !strings.Contains(got[0].Msg, tc.want) {
			t.Errorf("%s: clashes = %+v, want %q", name, got, tc.want)
		}
	}
}

func TestClashesAllowTheServersOfEachTargetAndOfTwoServersOfOne(t *testing.T) {
	lan := Text(
		"WS_TARGET", "lan",
		"WS_LAN", "10.88.0.0/24",
		"WS_LAN_ADDRESS", "10.88.0.10",
		"WS_REMOTE", "ws.example.com",
		"WS_REMOTE_PORT", "22022",
		"WS_TUNNEL_NETWORKS", "10.20.0.0/16",
		"WS_RESOLVERS", "a.test=10.20.0.53",
	)
	cloud := Text(
		"WS_SLOT", "1",
		"WS_TARGET", "cloud",
		"WS_REMOTE_PORT", "22",
		"WS_REMOTE", "198.51.100.7",
		"WS_TUNNEL_NETWORKS", "10.30.0.0/16",
		"WS_RESOLVERS", "b.test=10.30.0.53",
	)
	// Each second server holds its placed slot and ports: two machines behind one router and two cloud
	// VMs.
	lan2 := Text(
		"WS_SLOT", "3",
		"WS_TARGET", "lan",
		"WS_LAN", "10.88.0.0/24",
		"WS_LAN_ADDRESS", "10.88.0.11",
		"WS_REMOTE", "ws.example.com",
		"WS_TUNNEL_PORT", "51823",
		"WS_REMOTE_PORT", "22025",
		"WS_TUNNEL_NETWORKS", "10.21.0.0/16",
		"WS_RESOLVERS", "c.test=10.21.0.53",
	)
	cloud2 := Text(
		"WS_SLOT", "4",
		"WS_TARGET", "cloud",
		"WS_REMOTE_PORT", "22",
		"WS_REMOTE", "198.51.100.8",
		"WS_TUNNEL_NETWORKS", "10.31.0.0/16",
		"WS_RESOLVERS", "d.test=10.31.0.53",
	)
	for _, pair := range [][2]string{{lan, cloud}, {lan, lan2}, {cloud, cloud2}} {
		for _, order := range [][2]string{pair, {pair[1], pair[0]}} {
			if got := clashes(parse(t, order[0]), sibling(order[1])); len(got) != 0 {
				t.Errorf("%q beside %q: clashes = %+v, want none", order[0], order[1], got)
			}
		}
	}
}

// vpn returns a file's ikeward keys.
func vpn(dns, domains string) string {
	return Text("WS_IKEWARD_DNS", dns, "WS_IKEWARD_DOMAINS", domains, "WS_IKEWARD_FLOORS", "10.20.0.0/16 10.30.0.0/16")
}

func TestClashesRefuseTwoVPNsAndADomainThatTwoFilesAnswerApart(t *testing.T) {
	for name, tc := range map[string]struct{ own, theirs, want string }{
		"two profiles": {vpn("10.20.0.53", "a.test"), vpn("10.30.0.53", "b.test"),
			"WS_IKEWARD_DNS names a VPN, as server lab's does: ikeward serves one server's profile"},
		"own domain answered apart": {vpn("10.20.0.53", "a.test"), Text("WS_RESOLVERS", "a.test=10.0.0.1"),
			"WS_IKEWARD_DOMAINS's a.test is in server lab's WS_RESOLVERS, which lacks WS_IKEWARD_DNS=10.20.0.53"},
		"their domain answered apart": {Text("WS_RESOLVERS", "a.test=10.0.0.1"), vpn("10.20.0.53", "a.test"),
			"WS_RESOLVERS's a.test is in server lab's WS_IKEWARD_DOMAINS, whose WS_IKEWARD_DNS=10.20.0.53 it lacks"},
	} {
		got := clashes(parse(t, tc.own), sibling(tc.theirs))
		if len(got) != 1 || !strings.Contains(got[0].Msg, tc.want) {
			t.Errorf("%s: clashes = %+v, want %q", name, got, tc.want)
		}
	}
}

func TestClashesAllowADomainThatBothFilesAnswerAlike(t *testing.T) {
	vpnFile := vpn("10.20.0.53", "a.test")
	resolvers := Text("WS_RESOLVERS", "a.test=10.0.0.1,10.20.0.53")
	for _, pair := range [][2]string{{vpnFile, resolvers}, {resolvers, vpnFile}, {vpnFile, Text("WS_RESOLVERS", "b.test=10.0.0.1")}} {
		if got := clashes(parse(t, pair[0]), sibling(pair[1])); len(got) != 0 {
			t.Errorf("%q beside %q: clashes = %+v, want none", pair[0], pair[1], got)
		}
	}
}

func TestClashesIgnoreAServerWithNoKeys(t *testing.T) {
	if got := clashes(parse(t, Text("WS_SLOT", "1", "WS_TUNNEL_NETWORKS", "10.1.0.0/24", "WS_RESOLVERS", "a.test=10.0.0.1")), sibling("")); len(got) != 0 {
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
