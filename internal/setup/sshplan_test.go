package setup

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestSSHChangesOfNothing(t *testing.T) {
	f := sshFacts{Home: "/h", Stamp: "S"}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{
		Write{"/h/.ssh/config.d/ws", nil, []byte("new\n"), false, 0o600},
		Write{"/h/.ssh/config", nil, []byte("Include config.d/ws\n"), false, 0o600},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
}

func TestSSHChangesPutTheDirectoriesFirst(t *testing.T) {
	dirs := []step.Change{MakeDir{"/h/.ssh", 0o700}, MakeDir{"/h/.ssh/config.d", 0o700}}
	got := sshChanges(sshFacts{Home: "/h", Stamp: "S", Dirs: dirs}, []byte("new\n"))
	if len(got) != 4 || !reflect.DeepEqual(got[:2], dirs) {
		t.Errorf("sshChanges = %v, want the directories first", got)
	}
}

func TestSSHChangesOfModesAndInclude(t *testing.T) {
	f := sshFacts{
		Home: "/h", Stamp: "S",
		Ws:         path{Exists: true, Mode: 0o644, Text: []byte("new\n")},
		UserConfig: path{Exists: true, Mode: 0o640, Text: []byte("Host a\n")}, UserConfigNeedsInclude: true,
	}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{
		SetMode{"/h/.ssh/config.d/ws", 0o600},
		Backup{"/h/.ssh/config", "/h/.ssh/config.S~"},
		Write{"/h/.ssh/config", []byte("Host a\n"), []byte("Include config.d/ws\n\nHost a\n"), true, 0o640},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
	f.Ws.Mode, f.UserConfigNeedsInclude = 0o600, false
	if got := sshChanges(f, []byte("new\n")); len(got) != 0 {
		t.Errorf("sshChanges of a done setup = %v, want none", got)
	}
}
