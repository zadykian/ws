package conf

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestConfFormScopeShowsOneFilesKeys(t *testing.T) {
	f, _ := formOf(t, "", "")
	edits := map[string]string{"WS_TARGET": "cloud"}
	f.Scope = MacFile
	if got, want := keysOf(f.Fields(edits)), []string{"WS_GIT_NAME"}; !slices.Equal(got, want) {
		t.Errorf("mac.conf: keys = %v; want %v", got, want)
	}
	f.Scope = ServerFile
	got := keysOf(f.Fields(edits))
	if slices.Contains(got, "WS_GIT_NAME") || !slices.Contains(got, "WS_TARGET") || !slices.Contains(got, "WS_REMOTE") {
		t.Errorf("server file: keys = %v; want WS_TARGET and WS_REMOTE, and no WS_GIT_NAME", got)
	}
}

func TestConfFormScopeSavesToItsFileAlone(t *testing.T) {
	edits := map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "vm.example", "WS_GIT_NAME": "A B"}
	for scope, want := range map[Scope][2]string{
		MacFile:    {"WS_GIT_NAME=\"A B\"\nWS_GIT_EMAIL=a@b.dev\n", ""},
		ServerFile: {"", "WS_TARGET=cloud\nWS_REMOTE=vm.example\nWS_REMOTE_PORT=22\n"},
	} {
		f, m := formOf(t, "", "")
		f.Scope = scope
		if _, err := f.Save(context.Background(), &effecttest.Out{}, edits); err != nil {
			t.Fatal(err)
		}
		if mac, server := string(m.Files["/h/mac.conf"]), string(m.Files[serverOfTest]); mac != want[0] || server != want[1] {
			t.Errorf("scope %d: mac.conf = %q, server = %q; want %q", scope, mac, server, want)
		}
	}
}

func TestConfFormScopeSaysItsFileHoldsTheValues(t *testing.T) {
	f, _ := formOf(t, "", "")
	f.Scope = ServerFile
	out := &effecttest.Out{}
	if _, err := f.Save(context.Background(), out, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"say: " + serverOfTest + " holds these values already"}; !slices.Equal(out.Lines, want) {
		t.Errorf("lines = %q; want %q", out.Lines, want)
	}
}

func TestConfFormSaveChangesNeitherFileWhereOneRefuses(t *testing.T) {
	f, m := formOf(t, "", "WS_USER=ada\n")
	m.Links[serverOfTest] = "/dot/server.conf"
	_, err := f.Save(context.Background(), &effecttest.Out{}, map[string]string{"WS_GIT_NAME": "A B", "WS_TARGET": "cloud", "WS_REMOTE": "vm.example"})
	if err == nil {
		t.Fatal("Save passes a server file that is a link")
	}
	if mac, server := string(m.Files["/h/mac.conf"]), string(m.Files[serverOfTest]); mac != "" || server != "WS_USER=ada\n" {
		t.Errorf("mac.conf = %q, server = %q; want both as they were", mac, server)
	}
}
