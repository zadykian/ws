package values

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const macPath = "/h/.config/ws/values.yml"

func TestServerPathIsBesideMacConf(t *testing.T) {
	if got, want := ServerPath(macPath, "box"), "/h/.config/ws/servers/box/values.yml"; got != want {
		t.Errorf("ServerPath = %q, want %q", got, want)
	}
}

func TestLoadJoinsTheTwoFiles(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath] = []byte(Text("WS_GIT_SIGNING_PUB_KEY", "~/g.pub", "WS_IDENTITY_AGENT", "~/a.sock", "WS_IDENTITY_PUB_KEY", "~/k.pub"))
	m.Files[server] = []byte(Text(
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22",
	))
	c, err := Load(m, macPath, server)
	if err != nil {
		t.Fatal(err)
	}
	if c.GitSigningPubKey != "~/g.pub" || c.IdentityAgent != "~/a.sock" || c.IdentityPubKey != "~/k.pub" || c.Target != "cloud" {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadKeepsADefaultWhereAFileHasNoLine(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath], m.Files[server] = nil, nil
	c, err := Load(m, macPath, server)
	if err != nil || c.Target != "lan" {
		t.Errorf("Load = %+v, %v; want the defaults", c, err)
	}
}

func TestLoadRefusesEachFilesStrayKeyInItsOwnWords(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath] = []byte(Text("WS_TARGET", "lan"))
	m.Files[server] = []byte(Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"))
	_, err := Load(m, macPath, server)
	want := []string{
		macPath + ", line 1: WS_TARGET is a server's key: it goes in servers/NAME/values.yml, beside the Mac's values.yml",
		server + ", line 1: WS_GIT_SIGNING_PUB_KEY is the Mac's key: it goes in the Mac's values.yml",
	}
	if err == nil || err.Error() != strings.Join(want, "\n") {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestLoadSaysWhichFileIsMissing(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath] = nil
	_, err := Load(m, macPath, server)
	if want := server + " is missing: roles/client/README.md says what goes in it"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
	delete(m.Files, macPath)
	if _, err = Load(m, macPath, server); err == nil || !strings.HasPrefix(err.Error(), macPath+" is missing:") {
		t.Errorf("err = %v, want values.yml missing", err)
	}
}

func TestLoadRefusesANetworkThatAnotherServerRoutes(t *testing.T) {
	m := machineWith("ws", "lab")
	server := ServerPath(macPath, "ws")
	m.Files[macPath] = nil
	m.Files[server] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/24"))
	m.Files[ServerPath(macPath, "lab")] = []byte(Text("WS_TUNNEL_NETWORKS", "10.1.0.0/16"))
	_, err := Load(m, macPath, server)
	if want := server + ": WS_TUNNEL_NETWORKS's 10.1.0.0/24 overlaps server lab's"; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v, want %q first", err, want)
	}
}
