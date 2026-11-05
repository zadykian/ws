package host_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

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
