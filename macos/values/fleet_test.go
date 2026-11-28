package values

import (
	"errors"
	"strings"
	"testing"
)

// fleetOf returns the read of a machine with each server's file by name.
func fleetOf(files map[string]string) Fleet {
	m := machineWith()
	for name, text := range files {
		m.Dirs[serversDirPath+"/"+name] = 0o755
		m.Files[ServerPath(dirPath, name)] = []byte(text)
	}
	return ReadFleet(m, dirPath)
}

func TestFleetGivesEachServerItsRefusalAlone(t *testing.T) {
	slot := Text("WS_SLOT", "1")
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "0"), "box": slot, "lab": slot, "bad": Text("WS_SLOT", "999")})
	if got, want := strings.Join(f.Names(), " "), "bad box lab ws"; got != want {
		t.Errorf("Names = %q, want %q", got, want)
	}
	for _, name := range []string{"ws"} {
		if s, _ := f.Server(name); s.Err != nil {
			t.Errorf("%s is refused: %v", name, s.Err)
		}
	}
	box, _ := f.Server("box")
	if want := "WS_SLOT=1 is server lab's slot too"; box.Err == nil || !strings.Contains(box.Err.Error(), want) ||
		!strings.HasPrefix(box.Err.Error(), box.File+": ") {
		t.Errorf("box's refusal = %v, want %q in box's file's words", box.Err, want)
	}
	if bad, _ := f.Server("bad"); bad.Err == nil || bad.Conf.Target != "" {
		t.Errorf("bad = %+v, want a refusal and no values", bad)
	}
	if got := len(f.Accepted()); got != 1 {
		t.Errorf("Accepted holds %d servers, want ws alone", got)
	}
}

// TestFleetLeavesARefusedFileOutOfTheClashes: a bad file takes no slot from a good one.
func TestFleetLeavesARefusedFileOutOfTheClashes(t *testing.T) {
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "1"), "lab": Text("WS_SLOT", "1", "WS_REMOTE_PORT", "0")})
	if ws, _ := f.Server("ws"); ws.Err != nil {
		t.Errorf("ws is refused: %v", ws.Err)
	}
}

func TestFleetRefusesEachRefusedServerInTheSiblingsWords(t *testing.T) {
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "999")})
	err := f.Refused()
	if want := "server ws: " + ServerPath(dirPath, "ws") + ", line 1: WS_SLOT=999"; err == nil ||
		!strings.HasPrefix(err.Error(), want) || !strings.HasSuffix(err.Error(), BadFileHint("ws", nil)) {
		t.Errorf("Refused = %v, want %q first and the hint last", err, want)
	}
	if fleetOf(map[string]string{"ws": ""}).Refused() != nil {
		t.Error("Refused names a server that nothing refuses")
	}
}

// TestFleetHintsAtNoRemovalForAHostWithGuests: a host with guests cannot go, and no guest can while
// its host's file is refused, so the way out is its file alone.
func TestFleetHintsAtNoRemovalForAHostWithGuests(t *testing.T) {
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "999"), "g1": guestText, "g2": Text("WS_HOST", "ws", "WS_SLOT", "3")})
	err := f.Refused()
	if err == nil || !strings.Contains(err.Error(), "server ws: ") || !strings.HasSuffix(err.Error(), ": fix its file") {
		t.Errorf("Refused = %v, want ws's refusal and the hint to fix its file alone", err)
	}
	if err := f.Without("g1").Without("g2").Refused(); err == nil || !strings.HasSuffix(err.Error(), BadFileHint("ws", nil)) {
		t.Errorf("Refused with no guest left = %v, want the plain hint last", err)
	}
}

func TestFleetWithoutTakesTheServersClashesAway(t *testing.T) {
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "1"), "lab": Text("WS_SLOT", "1")})
	if f.Refused() == nil {
		t.Fatal("two servers at one slot are accepted")
	}
	if err := f.Without("ws").Refused(); err != nil {
		t.Errorf("Without(ws).Refused = %v, want none", err)
	}
	if got := f.Names(); len(got) != 2 {
		t.Errorf("Without changed the fleet: %v", got)
	}
}

func TestFleetGivesAGuestItsHostsWay(t *testing.T) {
	f := fleetOf(map[string]string{"ws": hostText, "g1": guestText})
	g1, _ := f.Server("g1")
	if g1.Err != nil || g1.Conf.Target != "guest" || g1.Conf.LAN != "10.88.0.0/23" || !g1.Conf.HostCable {
		t.Errorf("g1 = %+v, want the host's way", g1)
	}
	if got := f.GuestsOf("ws"); len(got) != 1 || got[0] != "g1" {
		t.Errorf("GuestsOf = %v, want g1", got)
	}
}

// TestFleetClashesAServerWithAGuestThroughItsHostsRemote: a guest answers at its host's address.
func TestFleetClashesAServerWithAGuestThroughItsHostsRemote(t *testing.T) {
	box := Text("WS_SLOT", "2", "WS_REMOTE", "ws.example.com", "WS_REMOTE_PORT", "22023", "WS_TUNNEL_PORT", "51821")
	f := fleetOf(map[string]string{"ws": hostText, "g1": guestText, "box": box})
	got, _ := f.Server("box")
	if want := "WS_TUNNEL_PORT=51821 is server g1's too"; got.Err == nil || !strings.Contains(got.Err.Error(), want) {
		t.Errorf("box = %v, want %q", got.Err, want)
	}
}

// TestFleetNamesAGuestWhoseHostIsRefusedAndKeepsItsHost: GuestsOf counts a guest that parses.
func TestFleetNamesAGuestWhoseHostIsRefusedAndKeepsItsHost(t *testing.T) {
	f := fleetOf(map[string]string{"ws": Text("WS_SLOT", "999"), "g1": guestText})
	g1, _ := f.Server("g1")
	if want := "WS_HOST=ws names a server whose file is refused"; g1.Err == nil || !strings.Contains(g1.Err.Error(), want) {
		t.Errorf("g1 = %v, want %q", g1.Err, want)
	}
	if got := f.GuestsOf("ws"); len(got) != 1 {
		t.Errorf("GuestsOf = %v, want g1", got)
	}
}

// TestFleetCountsNoServerAsItsOwnGuest: a file that names itself as its host is refused for it, and
// is no guest of itself.
func TestFleetCountsNoServerAsItsOwnGuest(t *testing.T) {
	f := fleetOf(map[string]string{"ws": hostText, "lab": Text("WS_HOST", "lab", "WS_SLOT", "1")})
	lab, _ := f.Server("lab")
	if want := "WS_HOST=lab names the guest's own server"; lab.Err == nil || !strings.Contains(lab.Err.Error(), want) {
		t.Errorf("lab = %v, want %q", lab.Err, want)
	}
	if got := f.GuestsOf("lab"); len(got) != 0 {
		t.Errorf("GuestsOf(lab) = %v, want none", got)
	}
}

func TestFleetLoadRefusesAServersBadFile(t *testing.T) {
	m := machineWith("ws")
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_TARGET", "cloud", "WS_REMOTE", "r.example", "WS_REMOTE_PORT", "22"))
	c, err := ReadFleet(m, dirPath).Load("ws")
	if err != nil || c.Target != "cloud" {
		t.Errorf("Load = %+v, %v", c, err)
	}
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_NOPE", "1"))
	_, err = ReadFleet(m, dirPath).Load("ws")
	if err == nil || !strings.Contains(err.Error(), "unknown key WS_NOPE") {
		t.Errorf("Load = %v, want the refusal of the file", err)
	}
	if _, err := ReadFleet(m, dirPath).Load("none"); err == nil {
		t.Error("Load named a server that is none")
	}
}

func TestFleetReadsNoServerBesideAMissingServersDir(t *testing.T) {
	m := machineWith()
	delete(m.Dirs, serversDirPath)
	if f := ReadFleet(m, dirPath); len(f.Servers) != 0 || f.NamesErr != nil || errors.Join(f.Refused()) != nil {
		t.Errorf("ReadFleet = %+v, want an empty fleet", f)
	}
}
