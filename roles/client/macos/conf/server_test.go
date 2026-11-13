package conf

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const macPath = "/h/.config/ws/mac.conf"

func TestServerPathIsBesideMacConf(t *testing.T) {
	if got, want := ServerPath(macPath, "box"), "/h/.config/ws/servers/box/server.conf"; got != want {
		t.Errorf("ServerPath = %q, want %q", got, want)
	}
}

func TestLoadJoinsTheTwoFiles(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath] = []byte("WS_GIT_NAME=\"Ada L\"\nWS_GIT_EMAIL=ada@example.com\nWS_IDENTITY_AGENT=~/a.sock\nWS_IDENTITY_PUB_KEY=~/k.pub\n")
	m.Files[server] = []byte("WS_TARGET=cloud\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22\nWS_USER=ada\n")
	c, err := Load(m, macPath, server)
	if err != nil {
		t.Fatal(err)
	}
	if c.GitName != "Ada L" || c.IdentityAgent != "~/a.sock" || c.IdentityPubKey != "~/k.pub" || c.Target != "cloud" || c.User != "ada" {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadKeepsADefaultWhereAFileHasNoLine(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath], m.Files[server] = nil, nil
	c, err := Load(m, macPath, server)
	if err != nil || c.Target != "lan" || c.User != "zadykian" {
		t.Errorf("Load = %+v, %v; want the defaults", c, err)
	}
}

func TestLoadRefusesEachFilesStrayKeyInItsOwnWords(t *testing.T) {
	m := effecttest.NewMachine()
	server := ServerPath(macPath, FirstServer)
	m.Files[macPath] = []byte("WS_TARGET=lan\n")
	m.Files[server] = []byte("WS_GIT_NAME=Ada\n")
	_, err := Load(m, macPath, server)
	want := []string{
		macPath + ", line 1: WS_TARGET is a server's key: it goes in servers/NAME/server.conf, beside mac.conf",
		server + ", line 1: WS_GIT_NAME is the Mac's key: it goes in mac.conf",
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
	if want := server + " is missing: roles/client/macos/README.md says what goes in it"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
	delete(m.Files, macPath)
	if _, err = Load(m, macPath, server); err == nil || !strings.HasPrefix(err.Error(), macPath+" is missing:") {
		t.Errorf("err = %v, want mac.conf missing", err)
	}
}
