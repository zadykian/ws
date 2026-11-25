package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

func TestUpdateSetupsFollowTheMiserc(t *testing.T) {
	m := effecttest.NewMachine()
	if args, err := UpdateSetups().Pick(m); err != nil || args != nil {
		t.Errorf("with no miserc: args = %v, err = %v; want no setup", args, err)
	}
	m.Files[setup.MisercPath] = setup.Miserc("guest")
	if args, err := UpdateSetups().Pick(m); err != nil || !reflect.DeepEqual(args, []string{"ubuntu", "guest", "setup"}) {
		t.Errorf("args = %v, err = %v; want the guest's setup", args, err)
	}
	m.Files[setup.MisercPath] = setup.Miserc("host")
	if args, err := UpdateSetups().Pick(m); err != nil || !reflect.DeepEqual(args, []string{"ubuntu", "host", "setup"}) {
		t.Errorf("args = %v, err = %v; want the host's setup", args, err)
	}
	for side, envs := range map[string][]string{
		"guest": {"guest", setup.SelfContained},
		"host":  {"host", setup.SelfContained},
	} {
		m.Files[setup.MisercPath] = setup.Miserc(envs...)
		if args, err := UpdateSetups().Pick(m); err != nil || !reflect.DeepEqual(args, []string{"ubuntu", side, "setup"}) {
			t.Errorf("with the mode: args = %v, err = %v; want the %s's setup", args, err, side)
		}
	}
	for _, envs := range [][]string{{"macos"}, {setup.SelfContained}, {"guest", "host"}, {"guest", "macos"}} {
		m.Files[setup.MisercPath] = setup.Miserc(envs...)
		if _, err := UpdateSetups().Pick(m); err == nil || !strings.Contains(err.Error(), "guest or host") {
			t.Errorf("envs %v: err = %v; want the envs refused", envs, err)
		}
	}
}

func TestUpdateSetupsNameBothSides(t *testing.T) {
	want := []string{"ws ubuntu guest setup", "ws ubuntu host setup"}
	if got := UpdateSetups().Names; !reflect.DeepEqual(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}
