package values

import (
	"strings"
	"testing"
)

func TestCheckPassesFilesThatHoldNoRefusal(t *testing.T) {
	m := machineWith("ws", "lab")
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_SLOT", "0"))
	m.Files[ServerPath(dirPath, "lab")] = []byte(Text("WS_SLOT", "1"))
	if err := Check(m, dirPath); err != nil {
		t.Errorf("Check = %v, want none", err)
	}
}

func TestCheckPassesAMacWithNoServerYet(t *testing.T) {
	m := machineWith()
	if err := Check(m, dirPath); err != nil {
		t.Errorf("Check = %v, want none", err)
	}
}

func TestCheckNamesEachBadKeyWithItsFileOnce(t *testing.T) {
	m := machineWith("ws", "lab")
	lab, ws := ServerPath(dirPath, "lab"), ServerPath(dirPath, "ws")
	m.Files[ws] = []byte(Text("WS_SLOT", "999"))
	m.Files[lab] = []byte(Text("WS_REMOTE_PORT", "0"))
	err := Check(m, dirPath)
	if err == nil {
		t.Fatal("Check passed files that hold bad keys")
	}
	for _, want := range []string{ws + ", line 1: WS_SLOT=999", lab + ", line 1: WS_REMOTE_PORT=0"} {
		if got := strings.Count(err.Error(), want); got != 1 {
			t.Errorf("Check = %q holds %q %d times, want once", err, want, got)
		}
	}
}

func TestCheckNamesAServerFileThatLacksItsPair(t *testing.T) {
	m := machineWith("ws")
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_LAN", "10.88.0.0/24"))
	err := Check(m, dirPath)
	if want := "WS_LAN and WS_LAN_ADDRESS go together"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Check = %v, want %q", err, want)
	}
}

func TestCheckRefusesWhatTwoServersShare(t *testing.T) {
	m := machineWith("ws", "lab")
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/24"))
	m.Files[ServerPath(dirPath, "lab")] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/16"))
	err := Check(m, dirPath)
	if want := "overlaps server lab's"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Check = %v, want %q", err, want)
	}
}

func TestCheckNamesTheServerAndTheWayOutOfItsFile(t *testing.T) {
	m := machineWith("ws", "lab")
	m.Files[ServerPath(dirPath, "ws")] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/24"))
	m.Files[ServerPath(dirPath, "lab")] = []byte(Text("WS_SLOT", "999"))
	err := Check(m, dirPath)
	if err == nil {
		t.Fatal("Check passed a bad file")
	}
	for _, want := range []string{"server lab: ", BadFileHint("lab", nil)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Check = %q, want it to contain %q", err, want)
		}
	}
}
