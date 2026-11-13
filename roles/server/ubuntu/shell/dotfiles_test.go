package shell

import (
	"bytes"
	"os"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

func TestDotfilesAreTheRolesFiles(t *testing.T) {
	m := effecttest.NewMachine()
	got, err := dotfiles(m, account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1, GID: 2}, "amd64")
	if err != nil || len(got) != 2 {
		t.Fatalf("got %d files, %v", len(got), err)
	}
	for i, name := range []string{"bashrc", "profile"} {
		want, err := os.ReadFile("testdata/" + name) //nolint:gosec // the name is one of the test's own constants
		if err != nil {
			t.Fatal(err)
		}
		if f := got[i].(file.File); !bytes.Equal(f.Content, want) {
			t.Errorf("%s differs from the role's rendering", name)
		}
	}
}

func TestDotnetArch(t *testing.T) {
	for arch, want := range map[string]string{"arm64": "arm64", "amd64": "x64", "riscv64": "x64"} {
		if got := dotnetArch(arch); got != want {
			t.Errorf("%s: %s, want %s", arch, got, want)
		}
	}
}
