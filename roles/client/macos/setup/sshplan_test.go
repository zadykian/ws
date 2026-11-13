package setup

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

var sshInclude = change.Write{Path: "/h/.ssh/config", Before: nil, After: []byte("Include config.d/ws\n"), Exists: false, Mode: 0o600}

func TestSSHChangesOfNothing(t *testing.T) {
	f := sshFacts{Home: "/h", Config: []step.Change{sshInclude}}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{change.Write{Path: "/h/.ssh/config.d/ws", Before: nil, After: []byte("new\n"), Exists: false, Mode: 0o600}, sshInclude}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
}

func TestSSHChangesPutTheDirectoriesFirst(t *testing.T) {
	dirs := []step.Change{change.MakeDir{Path: "/h/.ssh", Mode: 0o700}, change.MakeDir{Path: "/h/.ssh/config.d", Mode: 0o700}}
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
		change.SetMode{Path: "/h/.ssh/config.d/ws", Mode: 0o600},
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
