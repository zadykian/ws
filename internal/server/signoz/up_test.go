package signoz

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	psLine = "docker compose --project-name signoz ps --status running --services"
	upLine = "docker compose --project-directory /opt/signoz --project-name signoz up --detach --quiet-pull --remove-orphans"
)

// upFixture is a booted machine whose compose reports the named services as running.
func upFixture(_ *testing.T, running ...string) (*effecttest.Machine, Up) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Results[psLine] = effecttest.Result{Out: []byte(strings.Join(running, "\n") + "\n")}
	m.Results[upLine] = effecttest.Result{}
	m.Results[upLine+" --force-recreate"] = effecttest.Result{}
	return m, Up{M: m, Out: &effecttest.Out{}, Dir: "/opt/signoz"}
}

func services(t *testing.T) []string {
	got, err := Services(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestUpDoesNothingWhileTheServicesRun(t *testing.T) {
	_, u := upFixture(t, services(t)...)
	if p, err := u.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want none", p.Changes, err)
	}
}

func TestUpStartsWhereAServiceIsDown(t *testing.T) {
	m, u := upFixture(t, services(t)[1:]...)
	p, err := u.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if err := u.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "root "+upLine) {
		t.Errorf("calls %q: want up without a recreation", m.Calls)
	}
}

func TestUpRecreatesWhereTheMarkIsSetAndClearsIt(t *testing.T) {
	m, u := upFixture(t, services(t)...)
	m.Files["/opt/signoz/recreate"] = nil
	p, err := u.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if err := u.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "root "+upLine+" --force-recreate") {
		t.Errorf("calls %q: want up with a recreation", m.Calls)
	}
	if _, ok := m.Files["/opt/signoz/recreate"]; ok {
		t.Error("the mark is still set after an up")
	}
}

func TestUpSkipsWithoutSystemd(t *testing.T) {
	m, u := upFixture(t)
	delete(m.Dirs, "/run/systemd/system")
	out := u.Out.(*effecttest.Out)
	if p, err := u.Plan(t.Context()); err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("plan = %v, %v, said %q; want a skip said once", p.Changes, err, out.Lines)
	}
}

func TestUpFailsWhereComposeCannotListTheServices(t *testing.T) {
	m, u := upFixture(t)
	m.Results[psLine] = effecttest.Result{Err: errors.New("no daemon")}
	if _, err := u.Plan(t.Context()); err == nil {
		t.Error("plan succeeded, want the error of ps")
	}
}
