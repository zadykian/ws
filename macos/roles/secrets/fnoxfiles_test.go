package secrets_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

func TestCheckHostname(t *testing.T) {
	for _, h := range []string{"ws", "ws-a1", "1", strings.Repeat("a", 63)} {
		if err := secrets.CheckHostname(h); err != nil {
			t.Errorf("CheckHostname(%q) gives %v", h, err)
		}
	}
	for _, h := range []string{"nano", "backup", "local", "default", "Ws", "ws.lan", "-ws", "ws-", "ws_a", "", strings.Repeat("a", 64)} {
		if err := secrets.CheckHostname(h); err == nil || !strings.Contains(err.Error(), "set a hostname of one label at ws") {
			t.Errorf("CheckHostname(%q) gives %v, want the hint", h, err)
		}
	}
}

func TestHasSecrets(t *testing.T) {
	for text, want := range map[string]bool{
		"root = true\n": false,
		"[secrets]\n":   false,
		"[secrets]\nA= { provider = \"age\", value = \"YWdl\" }\n": true,
	} {
		if has, err := secrets.HasSecrets([]byte(text)); has != want || err != nil {
			t.Errorf("HasSecrets(%q) gives %v, %v, want %v", text, has, err, want)
		}
	}
}

func TestReadFilesSkipsTheLocalFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs[cloneDir] = 0o755
	for _, name := range []string{"fnox.toml", "fnox.ws-a.toml", "fnox.local.toml", "README.md", "fnox.toml.orig"} {
		m.Files[cloneDir+"/"+name] = []byte(name)
	}
	f, err := secrets.ReadFiles(m, cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if names := slices.Sorted(maps.Keys(f)); !slices.Equal(names, []string{"fnox.toml", "fnox.ws-a.toml"}) || string(f["fnox.toml"]) != "fnox.toml" {
		t.Errorf("ReadFiles reads %q, want fnox.toml and fnox.ws-a.toml", names)
	}
}

func TestWriteFilesWritesTheNamedFiles(t *testing.T) {
	m := effecttest.NewMachine()
	f := secrets.Files{"fnox.toml": []byte("a"), "fnox.ws-a.toml": []byte("b")}
	if err := secrets.WriteFiles(m, cloneDir, f, []string{"fnox.ws-a.toml"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Calls, []string{"write " + cloneDir + "/fnox.ws-a.toml 644"}) || string(m.Files[cloneDir+"/fnox.ws-a.toml"]) != "b" {
		t.Errorf("WriteFiles ran %q", m.Calls)
	}
	if err := secrets.WriteFiles(m, cloneDir, f, []string{"fnox.ws-b.toml"}); err == nil {
		t.Error("WriteFiles of a file it lacks gives no error")
	}
}
