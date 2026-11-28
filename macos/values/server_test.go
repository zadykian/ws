package values

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

// loadNamed returns the values of the server called name, as a command for one server reads them.
func loadNamed(m effect.Machine, name string) (Conf, error) { return ReadFleet(m, dirPath).Load(name) }

const dirPath = "/h/.config/ws"

func TestServerPathIsBelowTheDirectory(t *testing.T) {
	if got, want := ServerPath(dirPath, "box"), "/h/.config/ws/servers/box/values.yml"; got != want {
		t.Errorf("ServerPath = %q, want %q", got, want)
	}
}

func TestLoadReadsTheServersFile(t *testing.T) {
	m := machineWith("ws")
	server := ServerPath(dirPath, FirstServer)
	m.Files[server] = []byte(Text(
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22",
	))
	c, err := loadNamed(m, FirstServer)
	if err != nil {
		t.Fatal(err)
	}
	if c.Target != "cloud" || c.Remote != "r.example" || c.RemotePort != "22" {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadKeepsADefaultWhereAFileHasNoLine(t *testing.T) {
	m := machineWith("ws")
	server := ServerPath(dirPath, FirstServer)
	m.Files[server] = nil
	c, err := loadNamed(m, FirstServer)
	if err != nil || c.Target != "lan" {
		t.Errorf("Load = %+v, %v; want the defaults", c, err)
	}
}

func TestLoadRefusesANetworkThatAnotherServerRoutes(t *testing.T) {
	m := machineWith("ws", "lab")
	server := ServerPath(dirPath, "ws")
	m.Files[server] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/24"))
	m.Files[ServerPath(dirPath, "lab")] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/16"))
	_, err := loadNamed(m, "ws")
	if want := server + ": WS_TUNNEL_NETWORKS's 10.1.0.0/24 overlaps server lab's"; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v, want %q first", err, want)
	}
}
