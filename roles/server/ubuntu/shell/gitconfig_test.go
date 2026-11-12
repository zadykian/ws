package shell

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

func gitFixture(signing bool) (*effecttest.Machine, stdinMachine, *effecttest.Out, gitconfig) {
	m := effecttest.NewMachine()
	m.Results["git config --file - --name-only --list"] = effecttest.Result{}
	m.Results["git config --file - --list"] = effecttest.Result{}
	sm := stdinMachine{Machine: m, answers: map[string]string{}}
	out := &effecttest.Out{}
	a := host.Account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, sm, out, gitconfig{M: sm, Out: out, A: a, Name: "zadykian", Email: "me@zadykian.dev",
		Signing: signing, Key: "/home/ws/.ssh/id_ed25519_signing", Signers: "/home/ws/.config/git/allowed_signers"}
}

func TestGitconfigIsTheRolesRendering(t *testing.T) {
	for file, signing := range map[string]bool{"gitconfig_signing": true, "gitconfig_unsigned": false} {
		_, _, _, g := gitFixture(signing)
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
	m, _, out, g := gitFixture(true)
	m.Files["/home/ws/.gitconfig"] = []byte("[http]\n\textraheader = secret\n")
	got, err := g.Plan(context.Background())
	if err != nil || len(got) != 1 || strings.Contains(got[0].String(), "secret") || !strings.Contains(got[0].String(), "content not shown") {
		t.Fatalf("plan %v, %v", got, err)
	}
	if len(out.Lines) != 0 {
		t.Errorf("said %q for keys that git does not list", out.Lines)
	}
}

func TestGitconfigNamesWhatARewriteLoses(t *testing.T) {
	m, sm, out, g := gitFixture(true)
	m.Files["/home/ws/.gitconfig"] = []byte("OLD")
	m.Files["/home/ws/.config/git/local"] = []byte("LOCAL")
	sm.answers["OLD"] = "user.name\nhttp.https://x.y.extraheader\nuser.signingkey=/home/ws/other\n"
	sm.answers["LOCAL"] = "gpg.format\nuser.name\n"
	sm.answers["# Written"] = "user.name\n"
	if _, err := g.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(out.Lines, "\n")
	for _, want := range []string{"drops http.*.extraheader", "signing key, /home/ws/other,", "sets gpg.format"} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in %q", want, all)
		}
	}
}
