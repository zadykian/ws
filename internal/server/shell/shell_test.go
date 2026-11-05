package shell

import (
	"context"
	"slices"
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
