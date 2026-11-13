package setup

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

var sshInclude = Write{"/h/.ssh/config", nil, []byte("Include config.d/ws\n"), false, 0o600}

func TestSSHChangesOfNothing(t *testing.T) {
	f := sshFacts{Home: "/h", Config: []step.Change{sshInclude}}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{Write{"/h/.ssh/config.d/ws", nil, []byte("new\n"), false, 0o600}, sshInclude}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
}

func TestSSHChangesPutTheDirectoriesFirst(t *testing.T) {
	dirs := []step.Change{MakeDir{"/h/.ssh", 0o700}, MakeDir{"/h/.ssh/config.d", 0o700}}
	got := sshChanges(sshFacts{Home: "/h", Dirs: dirs, Config: []step.Change{sshInclude}}, []byte("new\n"))
	if len(got) != 4 || !reflect.DeepEqual(got[:2], dirs) {
		t.Errorf("sshChanges = %v, want the directories first", got)
	}
}

func TestSSHChangesOfModesAndInclude(t *testing.T) {
	f := sshFacts{
		Home: "/h", Ws: path{Exists: true, Mode: 0o644, Text: []byte("new\n")}, Config: []step.Change{sshInclude},
	}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{
		SetMode{"/h/.ssh/config.d/ws", 0o600},
		sshInclude,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
	f.Ws.Mode, f.Config = 0o600, nil
	if got := sshChanges(f, []byte("new\n")); len(got) != 0 {
		t.Errorf("sshChanges of a done setup = %v, want none", got)
	}
}
