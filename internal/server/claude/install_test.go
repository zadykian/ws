package claude

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

var root = host.User{Name: "root", Home: "/root"}

func TestInstallPlansWhatIsMissing(t *testing.T) {
	cases := []struct {
		have []string
		want []string
	}{
		{nil, []string{"install claude from https://claude.ai/install.sh",
			"install cld from https://github.com/zadykian/cld/releases/latest/download/install.sh"}},
		{[]string{"/root/.local/bin/claude"},
			[]string{"install cld from https://github.com/zadykian/cld/releases/latest/download/install.sh"}},
		{[]string{"/root/.local/bin/claude", "/root/.local/bin/cld"}, nil},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		for _, p := range c.have {
			m.Files[p] = nil
		}
		changes, err := Install{M: m, User: root}.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, ch := range changes {
			got = append(got, ch.String())
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("have %v: got %q, want %q", c.have, got, c.want)
		}
	}
}

func TestInstallRunsTheScriptAsTheUser(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["env HOME=/root curl -fsSL --retry 3 https://claude.ai/install.sh"] = effecttest.Result{Out: []byte("echo claude")}
	m.Results["env HOME=/root curl -fsSL --retry 3 https://github.com/zadykian/cld/releases/latest/download/install.sh"] =
		effecttest.Result{Out: []byte("echo cld")}
	m.Results["env HOME=/root env CLAUDE_INSTALL_ALLOW_SUDO=1 bash"] = effecttest.Result{}
	m.Results["env HOME=/root env CLD_INSTALL_DIR=/root/.local/bin sh"] = effecttest.Result{}
	changes, err := Install{M: m, User: root}.Plan(context.Background())
	if err != nil || len(changes) != 2 {
		t.Fatalf("%v, %v", changes, err)
	}
	for _, c := range changes {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"echo claude", "echo cld"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("scripts %q", m.Inputs)
	}
}
