package claude

import (
	"context"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const completionFile = "/root/.local/share/bash-completion/completions/cld"

func TestCompletionPlan(t *testing.T) {
	cases := []struct {
		name string
		have []byte // nil is no file
		want int
	}{
		{"missing", nil, 1},
		{"current", []byte("complete -F _cld cld\n"), 0},
		{"stale", []byte("old\n"), 1},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		m.Files["/root/.local/bin/cld"] = nil
		m.Results["env HOME=/root /root/.local/bin/cld completion bash"] = effecttest.Result{Out: []byte("complete -F _cld cld\n")}
		serveRead(m, root, completionFile, c.have)
		got, err := Completion{M: m, User: root}.Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %v, %v", c.name, got, err)
		}
	}
}

func TestCompletionWithoutCldPlansNothing(t *testing.T) {
	got, err := Completion{M: effecttest.NewMachine(), User: root}.Plan(context.Background())
	if err != nil || len(got) != 0 {
		t.Errorf("%v, %v", got, err)
	}
}

func TestCompletionRunsCldSetup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/root/.local/bin/cld"] = nil
	m.Results["env HOME=/root /root/.local/bin/cld completion bash"] = effecttest.Result{Out: []byte("script\n")}
	serveRead(m, root, completionFile, nil)
	m.Results["env HOME=/root /root/.local/bin/cld setup completion bash"] = effecttest.Result{}
	got, err := Completion{M: m, User: root}.Plan(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if err := got[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
