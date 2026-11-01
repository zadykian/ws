package otelcol

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestInstallRemovesAStrayDeb(t *testing.T) {
	m := machine(version + " ii\n")
	m.Files[debPath("amd64")] = []byte("deb")
	p, err := host.Lazy{Title: installStep, Parts: install(m, &effecttest.Units{})}.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || !strings.HasPrefix(p.Changes[0].String(), "remove ") {
		t.Errorf("plan with a stray .deb = %v, %v; want its removal", p.Changes, err)
	}
}
