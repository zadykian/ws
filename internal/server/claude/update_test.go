package claude

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestUpdatePlan(t *testing.T) {
	cases := []struct {
		name, have, tag, want string
	}{
		{"older", "cld 0.13.0", "v0.14.2", "update cld 0.13.0 to 0.14.2"},
		{"current", "cld 0.14.2", "v0.14.2", ""},
		{"newer than the release", "cld 0.15.0", "v0.14.2", ""},
		{"numeric order", "cld 0.9.0", "v0.10.0", "update cld 0.9.0 to 0.10.0"},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		m.Files["/root/.local/bin/cld"] = nil
		m.Results["env HOME=/root /root/.local/bin/cld --version"] = effecttest.Result{Out: []byte(c.have + "\n")}
		got, err := Update{M: m, User: root, URL: releases(t, c.tag)}.Plan(context.Background())
		switch {
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && len(got) != 0, c.want != "" && (len(got) != 1 || got[0].String() != c.want):
			t.Errorf("%s: %v, want %q", c.name, got, c.want)
		}
	}
}

func TestUpdateRefusesABuildFromSource(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/root/.local/bin/cld"] = nil
	m.Results["env HOME=/root /root/.local/bin/cld --version"] = effecttest.Result{Out: []byte("cld dev\n")}
	if _, err := (Update{M: m, User: root, URL: releases(t, "v1.0.0")}).Plan(context.Background()); err == nil {
		t.Error("a build from source is compared with a release")
	}
}

func TestUpdateWithoutCldPlansNothing(t *testing.T) {
	got, err := Update{M: effecttest.NewMachine(), User: root}.Plan(context.Background())
	if err != nil || len(got) != 0 {
		t.Errorf("%v, %v", got, err)
	}
}

func TestUpdateRunsCldUpdate(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/root/.local/bin/cld"] = nil
	m.Results["env HOME=/root /root/.local/bin/cld --version"] = effecttest.Result{Out: []byte("cld 0.1.0\n")}
	m.Results["env HOME=/root /root/.local/bin/cld update"] = effecttest.Result{}
	got, err := Update{M: m, User: root, URL: releases(t, "v0.2.0")}.Plan(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if err := got[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
