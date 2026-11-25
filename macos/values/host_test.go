package values

import (
	"strings"
	"testing"
)

// hostText is a host's file: on the LAN, behind a router, with the cable.
var hostText = Text(
	"WS_LAN", "10.88.0.0/23",
	"WS_LAN_ADDRESS", "10.88.1.10",
	"WS_REMOTE", "ws.example.com",
	"WS_CABLE_SERVICE", "USB LAN",
)

// guestText is a guest's file at slot 1, on the host ws.
var guestText = Text("WS_HOST", "ws", "WS_SLOT", "1", "WS_TUNNEL_PORT", "51821")

func TestLoadGivesAGuestItsHostsWay(t *testing.T) {
	m := machineWith("ws", "g1")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "ws")] = []byte(hostText)
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	c, err := Load(m, macPath, ServerPath(macPath, "g1"))
	if err != nil {
		t.Fatal(err)
	}
	if c.LAN != "10.88.0.0/23" || c.LANAddress != "10.88.1.10" || c.Remote != "ws.example.com" || !c.HostCable {
		t.Errorf("Load = %+v, want the host's LAN, address, remote and cable", c)
	}
	if c.Host != "ws" || c.Target != "guest" || c.CableService != "" || c.RemotePort != "" || c.TunnelPort() != "51821" {
		t.Errorf("Load = %+v, want a guest with no cable service of its own and its own port", c)
	}
}

func TestLoadKeepsAHostsOwnWay(t *testing.T) {
	m := machineWith("ws", "g1")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "ws")] = []byte(hostText)
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	c, err := Load(m, macPath, ServerPath(macPath, "ws"))
	if err != nil || c.Host != "" || c.HostCable || c.CableService != "USB LAN" || c.Target != "lan" {
		t.Errorf("Load = %+v, %v; want the host as its file gives it", c, err)
	}
}

func TestLoadRefusesAHostThatIsNoServer(t *testing.T) {
	m := machineWith("g1")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	_, err := Load(m, macPath, ServerPath(macPath, "g1"))
	want := ServerPath(macPath, "g1") + ": WS_HOST=ws names no server: " + ServerPath(macPath, "ws") + " is missing"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v, want %q first", err, want)
	}
}

func TestLoadRefusesAHostThatIsAGuest(t *testing.T) {
	m := machineWith("ws", "g1", "g2")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "ws")] = []byte(hostText)
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	m.Files[ServerPath(macPath, "g2")] = []byte(Text("WS_HOST", "g1", "WS_SLOT", "2", "WS_TUNNEL_PORT", "51822"))
	_, err := Load(m, macPath, ServerPath(macPath, "g2"))
	want := ServerPath(macPath, "g2") + ": WS_HOST=g1 names a guest: a host is a server with a way of its own"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v, want %q first", err, want)
	}
}

func TestParseRefusesAGuestsOwnWay(t *testing.T) {
	for name, keys := range map[string]string{
		"lan":     Text("WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.2"),
		"remote":  Text("WS_REMOTE", "a.b"),
		"port":    Text("WS_REMOTE", "a.b", "WS_REMOTE_PORT", "22022"),
		"cable":   Text("WS_CABLE_SERVICE", "USB LAN"),
		"cable p": Text("WS_CABLE_PORT", "en1"),
		"cloud":   Text("WS_TARGET", "cloud", "WS_REMOTE", "a.b", "WS_REMOTE_PORT", "22"),
	} {
		_, errs := ServerFile.Parse(Text("WS_HOST", "ws") + keys)
		if len(errs) != 1 || !strings.Contains(errs[0].Msg, "WS_HOST=ws names the host's way: a guest holds none of ") {
			t.Errorf("%s: %v, want a guest's refusal", name, errs)
		}
	}
}

func TestClashesBetweenAGuestAndItsHostFollowTheEndpoint(t *testing.T) {
	host := Conf{Remote: "ws.example.com", Port: "51820"}
	guest := Conf{Host: "ws", Remote: "ws.example.com", Slot: "1", Port: "51821"}
	other := Conf{Host: "ws", Remote: "ws.example.com", Slot: "2", Port: "51821"}
	sib := func(name string, c Conf) []Sibling { return []Sibling{{Name: name, Conf: c}} }
	// A guest holds no forward, so the default 22022 of each is no clash.
	if got := clashes(guest, sib("ws", host)); len(got) != 0 {
		t.Errorf("guest beside host: %+v, want none", got)
	}
	if got := clashes(host, sib("g1", guest)); len(got) != 0 {
		t.Errorf("host beside guest: %+v, want none", got)
	}
	// Two guests of one host share its router, so each takes a port of its own.
	got := clashes(other, sib("g1", guest))
	if len(got) != 1 || !strings.Contains(got[0].Msg, "WS_TUNNEL_PORT=51821 is server g1's too, behind WS_REMOTE ws.example.com") {
		t.Errorf("two guests: %+v, want the port's clash alone", got)
	}
	guest.Port = "51820"
	if got := clashes(guest, sib("ws", host)); len(got) != 1 {
		t.Errorf("guest on the host's port: %+v, want one clash", got)
	}
}

func TestClashesBetweenGuestsOfAHostWithNoRemoteFollowTheHost(t *testing.T) {
	host := Conf{Port: "51820"}
	guest := Conf{Host: "ws", Slot: "1", Port: "51821"}
	sib := func(name string, c Conf) []Sibling { return []Sibling{{Name: name, Conf: c}} }
	want := "WS_TUNNEL_PORT=51821 is server g1's too, on host ws"
	got := clashes(Conf{Host: "ws", Slot: "2", Port: "51821"}, sib("g1", guest))
	if len(got) != 1 || !strings.Contains(got[0].Msg, want) {
		t.Errorf("two guests: %+v, want %q", got, want)
	}
	got = clashes(Conf{Host: "ws", Slot: "1", Port: "51820"}, sib("ws", host))
	if len(got) != 1 || !strings.Contains(got[0].Msg, "WS_TUNNEL_PORT=51820 is server ws's too, on host ws") {
		t.Errorf("guest on the host's port: %+v, want one clash", got)
	}
	// Servers with no remote that are not of one host answer on their own addresses.
	if got := clashes(Conf{Port: "51821"}, sib("g1", guest)); len(got) != 0 {
		t.Errorf("a server beside another's guest: %+v, want none", got)
	}
	if got := clashes(Conf{Host: "ws", Port: "51822"}, sib("g1", guest)); len(got) != 0 {
		t.Errorf("guests on ports of their own: %+v, want none", got)
	}
}

func TestSiblingsResolveAGuestsHost(t *testing.T) {
	m := machineWith("ws", "g1", "box")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "ws")] = []byte(hostText)
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	got, err := Siblings(m, macPath, ServerPath(macPath, "box"))
	if err != nil || len(got) != 2 || got[1].Name != "ws" || got[0].Conf.Remote != "ws.example.com" {
		t.Errorf("Siblings = %+v, %v; want the guest with its host's remote", got, err)
	}
}

func TestACheckRefusesAGuestWhoseHostIsGone(t *testing.T) {
	m := machineWith("g1")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	err := Check(m, macPath)
	if err == nil || !strings.Contains(err.Error(), "server g1: ") || !strings.Contains(err.Error(), "WS_HOST=ws names no server") {
		t.Errorf("Check = %v, want g1's refusal", err)
	}
}

func TestLoadRefusesAGuestThatNamesItself(t *testing.T) {
	m := machineWith("g1")
	m.Files[macPath] = nil
	m.Files[ServerPath(macPath, "g1")] = []byte(Text("WS_HOST", "g1"))
	_, err := Load(m, macPath, ServerPath(macPath, "g1"))
	if err == nil || !strings.Contains(err.Error(), "WS_HOST=g1 names the guest's own server") {
		t.Errorf("err = %v, want the self-host refused", err)
	}
}

func TestGuestsOfListsTheServersThatNameTheHost(t *testing.T) {
	m := machineWith("ws", "g1", "g2", "box")
	m.Files[ServerPath(macPath, "g1")] = []byte(guestText)
	m.Files[ServerPath(macPath, "g2")] = []byte(Text("WS_HOST", "ws", "WS_SLOT", "2", "WS_TUNNEL_PORT", "51822"))
	m.Files[ServerPath(macPath, "box")] = []byte(Text("WS_HOST", "other"))
	got, err := GuestsOf(m, macPath, "ws")
	if err != nil || len(got) != 2 || got[0] != "g1" || got[1] != "g2" {
		t.Errorf("GuestsOf = %v, %v; want g1 and g2", got, err)
	}
}
