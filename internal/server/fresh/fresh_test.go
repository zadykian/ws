package fresh

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

func userMachine() (*effecttest.Machine, values.Values) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	return m, v
}

func TestConfigIsWrittenOnceForTheUser(t *testing.T) {
	m, v := userMachine()
	s := stepNamed(t, m, v, configStep)
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	changed, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false)
	if err != nil || changed {
		t.Errorf("the second run: changed %v, %v", changed, err)
	}
	path := "/home/ws/.config/fresh/config.json"
	if o := m.Owners[path]; o != [2]int{1000, 1000} {
		t.Errorf("%s owner %v", path, o)
	}
}
