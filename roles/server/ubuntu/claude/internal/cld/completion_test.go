package cld

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
)

const completionFile = "/root/.local/share/bash-completion/completions/cld"

// completionReadScript is the script that asuser.Read runs.
const completionReadScript = `if [ -e "$1" ]; then printf y; cat "$1"; else printf n; fi`

// serveCompletion fakes the command that asuser.Read runs for the user's file. A nil body is no file.
func serveCompletion(m *effecttest.Machine, u account.User, file string, body []byte) {
	out := []byte("n")
	if body != nil {
		out = append([]byte("y"), body...)
	}
	line := "env HOME=" + u.Home + " sh -c " + completionReadScript + " sh " + file
	if u.Name != "root" {
		line = "runuser -u " + u.Name + " -- env -C " + u.Home + " " + strings.TrimPrefix(line, "env ")
	}
	m.Results[line] = effecttest.Result{Out: out}
}

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
		serveCompletion(m, cldUser, completionFile, c.have)
		got, err := Completion{M: m, User: cldUser}.Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %v, %v", c.name, got, err)
		}
	}
}

func TestCompletionWithoutCldPlansNothing(t *testing.T) {
	got, err := Completion{M: effecttest.NewMachine(), User: cldUser}.Plan(context.Background())
	if err != nil || len(got) != 0 {
		t.Errorf("%v, %v", got, err)
	}
}

func TestCompletionRunsCldSetup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/root/.local/bin/cld"] = nil
	m.Results["env HOME=/root /root/.local/bin/cld completion bash"] = effecttest.Result{Out: []byte("script\n")}
	serveCompletion(m, cldUser, completionFile, nil)
	m.Results["env HOME=/root /root/.local/bin/cld setup completion bash"] = effecttest.Result{}
	got, err := Completion{M: m, User: cldUser}.Plan(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if err := got[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
