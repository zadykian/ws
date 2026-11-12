package host_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

// statFails is the fake machine, whose Stat fails at one path as a directory that ws cannot search
// would.
type statFails struct {
	*effecttest.Machine
	at string
}

func (m statFails) Stat(p string) (effect.Info, error) {
	if p == m.at {
		return effect.Info{}, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrPermission}
	}
	return m.Machine.Stat(p)
}

func TestRefuseLinksFailsWhereItCannotLook(t *testing.T) {
	m := statFails{effecttest.NewMachine(), "/home/ws/.config"}
	err := host.RefuseLinks(m, "/home/ws", "/home/ws/.config/x/f")
	if !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "at /home/ws/.config:") {
		t.Errorf("RefuseLinks = %v; want Stat's error, naming the path", err)
	}
}

func TestPlainRefusesALinkOnTheWay(t *testing.T) {
	for _, link := range []string{"/home/ws/.config", "/home/ws/.config/x", "/home/ws/.config/x/f"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		p := host.Plain{M: m, Home: "/home/ws", Paths: []string{"/home/ws/.config/x/f"}}
		if _, err := p.Plan(context.Background()); err == nil {
			t.Errorf("link at %s: no error", link)
		}
	}
}

func TestPlainPlansNothingWithoutALink(t *testing.T) {
	m := effecttest.NewMachine()
	p := host.Plain{M: m, Home: "/home/ws", Paths: []string{"/home/ws/.config/x/f"}}
	if got, err := p.Plan(context.Background()); err != nil || got != nil {
		t.Errorf("Plan = %v, %v; want none", got, err)
	}
}

func TestRefuseLinksChecksAPathOutsideHomeAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links["/srv"] = "/elsewhere"
	if err := host.RefuseLinks(m, "/home/ws", "/srv/f"); err != nil {
		t.Errorf("a link above a path outside home: %v", err)
	}
	m.Links["/srv/f"] = "/elsewhere"
	if err := host.RefuseLinks(m, "/home/ws", "/srv/f"); err == nil {
		t.Error("a link at a path outside home: no error")
	}
}
