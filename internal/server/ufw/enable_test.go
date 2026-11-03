package ufw_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/ufw"
)

func booted() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	return m
}

func planned(t *testing.T, r host.Resource) (got []string) {
	changes, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		got = append(got, c.String())
	}
	return got
}

func TestEnablePlansUntilTheConfSaysYes(t *testing.T) {
	m := booted()
	m.Files[ufw.ConfFile] = []byte("# on boot\nENABLED=no\nLOGLEVEL=low\n")
	e := ufw.Enable{M: m, Out: &effecttest.Out{}}
	if got, want := planned(t, e), []string{"ufw enable"}; !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	m.Files[ufw.ConfFile] = []byte("ENABLED=yes\n")
	m.Results["ufw status"] = effecttest.Result{Out: []byte("Status: active\n")}
	if got := planned(t, e); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestEnablePlansWhereTheRulesAreNotLoaded(t *testing.T) {
	m := booted()
	m.Files[ufw.ConfFile] = []byte("ENABLED=yes\n")
	m.Results["ufw status"] = effecttest.Result{Out: []byte("Status: inactive\n")}
	if got, want := planned(t, ufw.Enable{M: m, Out: &effecttest.Out{}}), []string{"ufw enable"}; !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}
