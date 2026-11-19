package secrets_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// keyFile is fnox's key file under the home /h.
const keyFile = "/h/.config/fnox/age.txt"

func TestKeyFileFollowsFnox(t *testing.T) {
	for _, tc := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"FNOX_AGE_KEY_FILE": "/k/age.txt", "FNOX_CONFIG_DIR": "/c", "HOME": "/h"}, "/k/age.txt"},
		{map[string]string{"FNOX_CONFIG_DIR": "/c", "XDG_CONFIG_HOME": "/x", "HOME": "/h"}, "/c/age.txt"},
		{map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/h"}, "/x/fnox/age.txt"},
		{map[string]string{"HOME": "/h"}, keyFile},
	} {
		got, err := secrets.KeyFile(func(k string) string { return tc.env[k] })
		if err != nil || got != tc.want {
			t.Errorf("KeyFile of %v gives %q, %v, want %s", tc.env, got, err, tc.want)
		}
	}
	for _, env := range []map[string]string{{"FNOX_CONFIG_DIR": "c", "HOME": "/h"}, {}} {
		if got, err := secrets.KeyFile(func(k string) string { return env[k] }); err == nil {
			t.Errorf("KeyFile of %v gives %q, want an error", env, got)
		}
	}
}

func TestKeepStubAddsItOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/.config/fnox"] = 0o700
	m.Files[keyFile] = []byte("# the user's\nAGE-PLUGIN-YUBIKEY-1OLD")
	wrote, err := secrets.KeepStub(m, keyFile, stubText)
	if err != nil || !wrote {
		t.Fatalf("KeepStub gives %v, %v, want a write", wrote, err)
	}
	want := "# the user's\nAGE-PLUGIN-YUBIKEY-1OLD\n" + stubText
	if got := string(m.Files[keyFile]); got != want || m.Modes[keyFile] != 0o600 {
		t.Errorf("the key file is %q, %o, want %q, 600", got, m.Modes[keyFile], want)
	}
	calls := len(m.Calls)
	if wrote, err := secrets.KeepStub(m, keyFile, stubText); err != nil || wrote {
		t.Errorf("KeepStub again gives %v, %v, want nothing written", wrote, err)
	}
	if slices.ContainsFunc(m.Calls[calls:], func(c string) bool { return strings.HasPrefix(c, "write") }) {
		t.Errorf("KeepStub again wrote: %q", m.Calls[calls:])
	}
}

func TestKeepStubMakesTheDirectory700(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h"] = 0o755
	if _, err := secrets.KeepStub(m, keyFile, stubText); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"/h/.config", "/h/.config/fnox"} {
		if m.Dirs[d] != 0o700 {
			t.Errorf("%s is %o, want 700", d, m.Dirs[d])
		}
	}
	if m.Dirs["/h"] != 0o755 || string(m.Files[keyFile]) != stubText {
		t.Errorf("the home is %o and the key file %q, want 755 and the stub alone", m.Dirs["/h"], m.Files[keyFile])
	}
}

func TestKeepStubRefusesAnSSHKeyFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[keyFile] = []byte("-----BEGIN OPENSSH PRIVATE KEY-----\n")
	if _, err := secrets.KeepStub(m, keyFile, stubText); err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Errorf("KeepStub of an SSH key file gives %v, want an error", err)
	}
	if m.Modes[keyFile] != 0 {
		t.Errorf("KeepStub wrote the SSH key file %o", m.Modes[keyFile])
	}
}

func TestKeepStubRefusesAStubWithoutItsLine(t *testing.T) {
	m := effecttest.NewMachine()
	if _, err := secrets.KeepStub(m, keyFile, "# no stub\n"); err == nil {
		t.Error("KeepStub takes an identity file without a stub line")
	}
	if _, ok := m.Files[keyFile]; ok {
		t.Errorf("KeepStub wrote %s", keyFile)
	}
}

func TestFnoxEnvPutsUsrLocalBinFirst(t *testing.T) {
	env := map[string]string{"PATH": "/usr/bin:/bin"}
	got := secrets.FnoxEnv(func(k string) string { return env[k] }, keyFile)
	want := []string{"PATH=/usr/local/bin:/usr/bin:/bin", "FNOX_AGE_KEY_FILE=" + keyFile}
	if !slices.Equal(got, want) {
		t.Errorf("FnoxEnv gives %q, want %q", got, want)
	}
	if got := secrets.FnoxEnv(func(string) string { return "" }, keyFile); got[0] != "PATH=/usr/local/bin" {
		t.Errorf("FnoxEnv without PATH gives %q, want /usr/local/bin alone", got[0])
	}
}
