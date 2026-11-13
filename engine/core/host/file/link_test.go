package file_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/file"
)

func TestLink(t *testing.T) {
	m := effecttest.NewMachine()
	l := file.Link{M: m, Path: "/usr/local/bin/x", Target: "/opt/x"}
	if got := applyAll(t, l); len(got) != 1 || got[0] != "link /usr/local/bin/x to /opt/x" {
		t.Errorf("first plan = %q, want a link", got)
	}
	if got := applyAll(t, l); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	m.Links[l.Path] = "/opt/old"
	if got := applyAll(t, l); len(got) != 1 || m.Links[l.Path] != "/opt/x" {
		t.Errorf("a wrong link: plan %q, link %q; want it replaced", got, m.Links[l.Path])
	}
}
