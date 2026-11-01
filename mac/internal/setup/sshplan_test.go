package setup

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/mac/internal/step"
)

func TestSSHChangesOfNothing(t *testing.T) {
	f := sshFacts{Home: "/h", Stamp: "S"}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{
		MakeDir{"/h/.ssh", 0o700},
		MakeDir{"/h/.ssh/config.d", 0o700},
		Write{"/h/.ssh/config.d/ws", nil, []byte("new\n"), false, 0o600},
		Write{"/h/.ssh/config", nil, []byte("Include config.d/ws\n"), false, 0o600},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
}

func TestSSHChangesOfModesAndInclude(t *testing.T) {
	f := sshFacts{
		Home: "/h", Stamp: "S", Dir: path{Exists: true, Mode: 0o700},
		ConfigDir:  path{Exists: true, Mode: 0o755},
		Ws:         path{Exists: true, Mode: 0o644, Text: []byte("new\n")},
		UserConfig: path{Exists: true, Mode: 0o640, Text: []byte("Host a\n")}, UserConfigNeedsInclude: true,
	}
	got := sshChanges(f, []byte("new\n"))
	want := []step.Change{
		SetMode{"/h/.ssh/config.d", 0o700},
		SetMode{"/h/.ssh/config.d/ws", 0o600},
		Backup{"/h/.ssh/config", "/h/.ssh/config.S~"},
		Write{"/h/.ssh/config", []byte("Host a\n"), []byte("Include config.d/ws\n\nHost a\n"), true, 0o640},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshChanges = %v, want %v", got, want)
	}
	f.Ws.Mode, f.ConfigDir.Mode, f.UserConfigNeedsInclude = 0o600, 0o700, false
	if got := sshChanges(f, []byte("new\n")); len(got) != 0 {
		t.Errorf("sshChanges of a done setup = %v, want none", got)
	}
}
