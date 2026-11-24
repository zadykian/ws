package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestMiserc(t *testing.T) {
	var rc struct {
		Env  []string            `toml:"env"`
		Sets map[string][]string `toml:"env_sets"`
	}
	if err := toml.Unmarshal(Miserc("guest"), &rc); err != nil {
		t.Fatal(err)
	}
	if strings.Join(rc.Env, ",") != "ws" || strings.Join(rc.Sets["ws"], ",") != "guest" || len(rc.Sets) != 1 {
		t.Errorf("miserc = %+v", rc)
	}
}

func TestConfigTrustsTheTree(t *testing.T) {
	var c struct {
		Settings struct {
			Trusted []string `toml:"trusted_config_paths"`
		} `toml:"settings"`
	}
	if err := toml.Unmarshal(Config(), &c); err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Settings.Trusted, ",") != TreeDir {
		t.Errorf("trusted_config_paths = %v", c.Settings.Trusted)
	}
}

func TestSystemMiseWritesRootsFilesOnce(t *testing.T) {
	m := effecttest.NewMachine()
	for _, r := range SystemMise(m, "guest") {
		changes, err := r.Plan(context.Background())
		if err != nil || len(changes) != 1 {
			t.Fatalf("plan = %v, %v", changes, err)
		}
		if err := changes[0].Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if m.Dirs["/etc/mise"] != 0o755 {
		t.Errorf("/etc/mise = %o, want it made at 755", m.Dirs["/etc/mise"])
	}
	for _, r := range SystemMise(m, "guest") {
		if changes, err := r.Plan(context.Background()); err != nil || len(changes) != 0 {
			t.Errorf("a second plan = %v, %v", changes, err)
		}
	}
}

func TestSideEnvsReadsTheMiserc(t *testing.T) {
	m := effecttest.NewMachine()
	envs, err := SideEnvs(m)
	if err != nil || len(envs) != 0 {
		t.Errorf("with no miserc: envs = %v, err = %v; want none", envs, err)
	}
	m.Files[MisercPath] = Miserc("guest")
	if envs, err = SideEnvs(m); err != nil || strings.Join(envs, ",") != "guest" {
		t.Errorf("envs = %v, err = %v; want guest", envs, err)
	}
	m.Files[MisercPath] = []byte("env = [")
	if _, err = SideEnvs(m); err == nil || !strings.Contains(err.Error(), MisercPath) {
		t.Errorf("err = %v; want the miserc named", err)
	}
}
