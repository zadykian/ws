package devtools

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestRustupInstallsOnce(t *testing.T) {
	for _, user := range []string{"root", "ws"} {
		m := effecttest.NewMachine()
		home := "/home/" + user
		install := "bash -c " + installRustup
		m.Results["env HOME="+home+" "+install] = effecttest.Result{}
		m.Results["runuser -u ws -- env -C "+home+" HOME="+home+" "+install] = effecttest.Result{}
		r := Rustup{M: m, User: user, Home: home}
		changes, err := r.Plan(context.Background())
		if err != nil || len(changes) != 1 {
			t.Fatalf("%s: got %d changes, %v", user, len(changes), err)
		}
		if err := changes[0].Apply(context.Background()); err != nil {
			t.Fatalf("%s: %v", user, err)
		}
		last := m.Calls[len(m.Calls)-1]
		if !strings.Contains(last, "HOME="+home) || strings.HasPrefix(last, "root ") != (user != "root") {
			t.Errorf("%s: ran %q", user, last)
		}
		m.Files[home+"/.cargo/bin/rustup"] = []byte("x")
		if changes, err := r.Plan(context.Background()); err != nil || len(changes) != 0 {
			t.Errorf("%s: a second plan has %v, %v", user, changes, err)
		}
	}
}
