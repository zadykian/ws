package lazygit

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

func steps() []step.Step {
	return Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Out{})
}

func TestStepsAreTheRole(t *testing.T) {
	var names []string
	for _, s := range steps() {
		names = append(names, s.Name())
	}
	if want := []string{packageStep, wrapperStep, configStep}; !slices.Equal(names, want) {
		t.Errorf("steps %v, want %v", names, want)
	}
}

func TestTheWrapperWaitsOnThePackage(t *testing.T) {
	if got := steps()[1].WaitsOn(); got != packageStep {
		t.Errorf("the wrapper waits on %q", got)
	}
}

const configPath = "/home/ws/.config/lazygit/config.yml"

// configOn returns the config's step for the user ws on a machine that has the user's home.
func configOn() (*effecttest.Machine, step.Step) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	return m, Steps(v, secrets.Secrets{}, m, &effecttest.Out{})[2]
}

func TestConfigIsWrittenOnceForTheUser(t *testing.T) {
	m, s := configOn()
	if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	changed, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false)
	if err != nil || changed {
		t.Errorf("the second run: changed %v, %v", changed, err)
	}
	if o := m.Owners[configPath]; o != [2]int{1000, 1000} {
		t.Errorf("owner %v", o)
	}
	if !bytes.Equal(m.Files[configPath], config) {
		t.Error("config.yml differs from the embedded file")
	}
}

func TestConfigRefusesALink(t *testing.T) {
	for _, link := range []string{"/home/ws/.config", "/home/ws/.config/lazygit", configPath} {
		m, s := configOn()
		m.Links[link] = "/srv/target"
		if _, err := step.Run(context.Background(), []step.Step{s}, &effecttest.Out{}, false); err == nil {
			t.Errorf("%s: no error", link)
		}
		if len(m.Owners) != 0 || len(m.Files) != 0 {
			t.Errorf("%s: changed %v %v", link, m.Owners, m.Files)
		}
	}
}

func TestTheConfigStartsWithItsHeader(t *testing.T) {
	if !bytes.HasPrefix(config, []byte("# Written by github.com/zadykian/ws,")) {
		t.Error("config.yml lacks the header")
	}
}
