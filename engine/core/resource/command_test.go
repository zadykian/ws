package resource_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

func TestCommandShowsItsLine(t *testing.T) {
	tests := []struct {
		c    resource.Command
		want string
		kind step.ChangeKind
	}{
		{resource.Command{Name: "brew", Args: []string{"install", "gh"}, Of: step.Install},
			"would run brew install gh", step.Install},
		{resource.Command{Name: "/usr/bin/open", Label: "open it"}, "would run open it", step.OtherChange},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want || tt.c.Kind() != tt.kind {
			t.Errorf("%+v: %q, %s; want %q, %s", tt.c, got, tt.c.Kind(), tt.want, tt.kind)
		}
	}
}

func TestCommandRunsAsRootOrNot(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["brew list"] = effecttest.Result{}
	m.Results["launchctl list"] = effecttest.Result{Err: errors.New("boom")}
	if err := (resource.Command{M: m, Name: "brew", Args: []string{"list"}}).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	err := resource.Command{M: m, Root: true, Name: "launchctl", Args: []string{"list"}}.Apply(context.Background())
	if err == nil || err.Error() != "boom" {
		t.Errorf("Apply = %v; want the bare cause, boom", err)
	}
	if got := strings.Join(m.Calls, ";"); got != "run brew list;root launchctl list" {
		t.Errorf("ran %q; want brew as the user and launchctl as root", got)
	}
}
