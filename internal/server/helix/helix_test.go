package helix

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// stepNamed returns the role's step of that name.
func stepNamed(t *testing.T, m *effecttest.Machine, v values.Values, name string) step.Step {
	for _, s := range Steps(v, secrets.Secrets{}, m, &effecttest.Out{}) {
		if s.Name() == name {
			return s
		}
	}
	t.Fatalf("no step %q", name)
	return nil
}

func TestConfigIsWrittenOnceForTheUser(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	s := stepNamed(t, m, v, "Helix config")
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if o := m.Owners["/home/ws/.config/helix/config.toml"]; o != [2]int{1000, 1000} {
		t.Errorf("config.toml owner %v", o)
	}
	if _, ok := m.Files["/home/ws/.config/helix/languages.toml"]; !ok {
		t.Error("languages.toml is not written")
	}
}

func TestConfigRefusesALink(t *testing.T) {
	for _, link := range []string{"/home/ws/.config", "/home/ws/.config/helix", "/home/ws/.config/helix/config.toml", "/home/ws/.config/helix/.ws-grammars"} {
		m := effecttest.NewMachine()
		m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
		m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
		m.Links[link] = "/srv/target"
		v := values.Default()
		v.User, v.Home = "ws", "/home/ws"
		s := stepNamed(t, m, v, "Helix config")
		if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err == nil {
			t.Errorf("%s: no error", link)
		}
		if len(m.Owners) != 0 || len(m.Files) != 0 {
			t.Errorf("%s: changed %v %v", link, m.Owners, m.Files)
		}
	}
}
