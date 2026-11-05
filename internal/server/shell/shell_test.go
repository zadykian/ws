package shell

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

func stepNames(v values.Values, s secrets.Secrets) []string {
	var names []string
	for _, st := range Steps(v, s, effecttest.NewMachine(), &effecttest.Out{}) {
		names = append(names, st.Name())
	}
	return names
}

func TestStepsAreTheRolesInOrder(t *testing.T) {
	got := stepNames(values.Default(), secrets.Secrets{})
	if want := []string{"Shell links", "Shell packages", "Shell files", keyStep, "Shell allowed signers", "Shell git config", "Shell secret env", "Shell git tokens", "Shell clones"}; !slices.Equal(got, want) {
		t.Errorf("steps = %v, want %v", got, want)
	}
	v := values.Default()
	v.Shell.GitSigning = false
	if got := stepNames(v, secrets.Secrets{}); slices.Contains(got, keyStep) {
		t.Errorf("steps without signing = %v, want no key", got)
	}
}

func TestALinkKeepsAReportFromThePathsBehindIt(t *testing.T) {
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Links["/home/ws/.bashrc"] = "/etc/shadow"
	m.Files["/home/ws/.bashrc"] = []byte("root:$6$hash:19000\n")
	reports, err := step.Read(t.Context(), Steps(v, secrets.Secrets{}, m, &effecttest.Out{}))
	if err != nil {
		t.Fatal(err)
	}
	if text := fmt.Sprintf("%+v", reports); strings.Contains(text, "$6$hash") {
		t.Errorf("reports = %s, want nothing read through the link", text)
	}
	if got := reports[0].Items; len(got) != 1 || !strings.Contains(got[0].Conflict, "/etc/shadow") {
		t.Errorf("items = %+v, want the refusal", got)
	}
}

func TestLinksStepRefusesALinkAtAFileRootWrites(t *testing.T) {
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	v.GitSigningKey = "/home/ws/.ssh/id_signing"
	for _, link := range []string{"/home/ws/.bashrc", "/home/ws/.ssh/id_signing.pub", "/home/ws/.secrets", "/home/ws/.config/git"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		first := Steps(v, secrets.Secrets{}, m, &effecttest.Out{})[0]
		if _, err := step.Run(context.Background(), []step.Step{first}, &effecttest.Out{}, false); err == nil {
			t.Errorf("a link at %s: no error", link)
		}
	}
}
