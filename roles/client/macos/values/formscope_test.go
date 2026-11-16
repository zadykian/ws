package values

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
	if got, want := keysOf(f.Fields(edits)), []string(nil); !slices.Equal(got, want) {
		t.Errorf("values.yml: keys = %v; want %v", got, want)
	}
	f.Scope = ServerFile
	got := keysOf(f.Fields(edits))
	if !slices.Contains(got, "WS_TARGET") || !slices.Contains(got, "WS_REMOTE") {
		t.Errorf("server file: keys = %v; want WS_TARGET and WS_REMOTE", got)
	}
}

func TestConfFormScopeSavesToItsFileAlone(t *testing.T) {
	edits := map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "vm.example"}
	for scope, want := range map[Scope][2]string{
		MacFile:    {"", ""},
		ServerFile: {"", Text("WS_TARGET", "cloud", "WS_REMOTE", "vm.example", "WS_REMOTE_PORT", "22")},
	} {
		f, m := formOf(t, "", "")
		f.Scope = scope
		if _, err := f.Save(context.Background(), &effecttest.Out{}, edits); err != nil {
			t.Fatal(err)
		}
		if mac, server := string(m.Files["/h/values.yml"]), string(m.Files[serverOfTest]); mac != want[0] || server != want[1] {
			t.Errorf("scope %d: values.yml = %q, server = %q; want %q", scope, mac, server, want)
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
	f, m := formOf(t, "", Text("WS_USER", "ada"))
	m.Links[serverOfTest] = "/dot/values.yml"
	_, err := f.Save(context.Background(), &effecttest.Out{}, map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "vm.example"})
	if err == nil {
		t.Fatal("Save passes a server file that is a link")
	}
	if mac, server := string(m.Files["/h/values.yml"]), string(m.Files[serverOfTest]); mac != "" || server != Text("WS_USER", "ada") {
		t.Errorf("values.yml = %q, server = %q; want both as they were", mac, server)
	}
}
