package shell

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func gitFixture(signing bool) (*effecttest.Machine, *effecttest.Out, gitconfig) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	a := account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, out, gitconfig{M: m, Out: out, A: a, Name: "zadykian", Email: "me@zadykian.dev",
		Signing: signing, Key: "/home/ws/.ssh/id_ed25519_signing", Signers: "/home/ws/.config/git/allowed_signers"}
}

func TestGitconfigIsTheRolesRendering(t *testing.T) {
	for file, signing := range map[string]bool{"gitconfig_signing": true, "gitconfig_unsigned": false} {
		_, _, g := gitFixture(signing)
		want, err := os.ReadFile("testdata/" + file) //nolint:gosec // the name is one of the test's own constants
		if err != nil {
			t.Fatal(err)
		}
		if got, err := g.content(); err != nil || string(got) != string(want) {
			t.Errorf("%s: differs from the role's rendering (%v)", file, err)
		}
	}
}

func TestGitconfigShowsNoContent(t *testing.T) {
	m, out, g := gitFixture(true)
	m.Files["/home/ws/.gitconfig"] = []byte("[http]\n\textraheader = secret\n")
	got, err := g.Plan(context.Background())
	if err != nil || len(got) != 1 || strings.Contains(got[0].String(), "secret") || !strings.Contains(got[0].String(), "content not shown") {
		t.Fatalf("plan %v, %v", got, err)
	}
	if len(out.Lines) != 0 {
		t.Errorf("said %q for keys that git does not list", out.Lines)
	}
}
