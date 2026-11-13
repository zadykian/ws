package conf

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const serversDirPath = "/h/.config/ws/servers"

// machineWith returns a machine whose servers/ holds a server.conf in each directory named.
func machineWith(names ...string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs[serversDirPath] = 0o755
	for _, n := range names {
		m.Dirs[serversDirPath+"/"+n] = 0o755
		m.Files[ServerPath(macPath, n)] = nil
	}
	return m
}

func TestIsName(t *testing.T) {
	for s, want := range map[string]bool{
		"ws": true, "a": true, "box-2": true, "abcdefghijkl": true,
		"": false, "Lab": false, "1box": false, "-box": false, "box_2": false, "abcdefghijklm": false,
	} {
		if got := IsName(s); got != want {
			t.Errorf("IsName(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestServersListsTheDirectoriesThatHoldAServerFile(t *testing.T) {
	m := machineWith("ws", "box")
	m.Dirs[serversDirPath+"/.git"] = 0o755
	m.Files[ServerPath(macPath, ".git")] = nil
	m.Dirs[serversDirPath+"/empty"] = 0o755
	m.Files[serversDirPath+"/notes"] = nil
	got, err := Servers(m, macPath)
	if err != nil || strings.Join(got, ",") != "box,ws" {
		t.Errorf("Servers = %v, %v; want box, ws", got, err)
	}
}

func TestServersFindsNoneWithNoDirectory(t *testing.T) {
	got, err := Servers(effecttest.NewMachine(), macPath)
	if err != nil || len(got) != 0 {
		t.Errorf("Servers = %v, %v; want none", got, err)
	}
}

func TestServersRefusesABadNameThatHoldsAServerFile(t *testing.T) {
	_, err := Servers(machineWith("Lab"), macPath)
	want := serversDirPath + "/Lab is not a server's name: " + NameForm + ": rename it"
	if err == nil || err.Error() != want {
		t.Errorf("Servers error = %v, want %q", err, want)
	}
	m := machineWith("ws")
	m.Dirs[serversDirPath+"/Lab"] = 0o755
	if got, err := Servers(m, macPath); err != nil || len(got) != 1 {
		t.Errorf("Servers = %v, %v; a bad name with no server.conf is not a server", got, err)
	}
}
