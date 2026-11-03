package devtools

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	readCall  = "runuser -u ws -- env HOME=/home/ws gh auth token -h github.com"
	loginCall = "runuser -u ws -- env HOME=/home/ws gh auth login -h github.com -p https --insecure-storage --with-token"
)

func ghStep(m *effecttest.Machine, token string) GHLogin {
	return GHLogin{M: m, User: "ws", Home: "/home/ws", Token: token}
}

func TestGHLoginPlan(t *testing.T) {
	tests := []struct {
		name, token string
		held        effecttest.Result
		want        int
	}{
		{"the same token", "MY_TOKEN", effecttest.Result{Out: []byte("MY_TOKEN\n")}, 0},
		{"another token", "MY_TOKEN", effecttest.Result{Out: []byte("OTHER\n")}, 1},
		{"no login", "MY_TOKEN", effecttest.Result{Err: errors.New("exit 1")}, 1},
		{"no token in the file", "", effecttest.Result{Out: []byte("OTHER\n")}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[readCall] = tt.held
			changes, err := ghStep(m, tt.token).Plan(context.Background())
			if err != nil || len(changes) != tt.want {
				t.Fatalf("got %v, %v; want %d changes", changes, err, tt.want)
			}
			for _, c := range changes {
				if c.String() != "log gh in by token.github" || strings.Contains(c.String(), tt.token) {
					t.Errorf("change %q", c)
				}
			}
		})
	}
}

func TestGHLoginAppliesWithTheTokenOnStdin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[readCall] = effecttest.Result{Err: errors.New("exit 1")}
	m.Results[loginCall] = effecttest.Result{}
	changes, _ := ghStep(m, "MY_TOKEN").Plan(context.Background())
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Inputs, []string{"MY_TOKEN"}) {
		t.Errorf("inputs %q", m.Inputs)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "MY_TOKEN") {
			t.Errorf("the token is in a call: %q", c)
		}
	}
}

func TestGHLoginRefusalNamesNoToken(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[readCall] = effecttest.Result{Err: errors.New("exit 1")}
	m.Results[loginCall] = effecttest.Result{Err: errors.New("gh refused")}
	changes, _ := ghStep(m, "MY_TOKEN").Plan(context.Background())
	err := changes[0].Apply(context.Background())
	if err == nil || strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("got %v", err)
	}
}
