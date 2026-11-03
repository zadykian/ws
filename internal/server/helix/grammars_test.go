package helix

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const (
	hxVersion = "env HOME=/root hx --version"
	stampFile = "/root/.config/helix/.ws-grammars"
)

func newGrammars(m *effecttest.Machine) Grammars {
	m.Files[hxBin] = nil
	m.Results[hxVersion] = effecttest.Result{Out: []byte("helix 25.07.1 (abc)\n")}
	return Grammars{M: m, User: host.User{Name: "root", Home: "/root"}, Names: []string{"go", "xml"}}
}

func withLibs(m *effecttest.Machine, g Grammars) {
	for _, n := range g.Names {
		m.Files[g.libPath(n)] = nil
	}
}

func TestGrammarsPlan(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m *effecttest.Machine, g Grammars)
		want  string
	}{
		{"no stamp", func(m *effecttest.Machine, g Grammars) { withLibs(m, g) },
			"fetch and build helix's grammars: no earlier run"},
		{"current", func(m *effecttest.Machine, g Grammars) {
			withLibs(m, g)
			m.Files[stampFile] = []byte("helix 25.07.1 (abc)\ngo\nxml\n")
		}, ""},
		{"new hx", func(m *effecttest.Machine, g Grammars) {
			withLibs(m, g)
			m.Files[stampFile] = []byte("helix 24.0 (abc)\ngo\nxml\n")
		}, "fetch and build helix's grammars: hx or the grammar list changed"},
		{"new list", func(m *effecttest.Machine, g Grammars) {
			withLibs(m, g)
			m.Files[stampFile] = []byte("helix 25.07.1 (abc)\ngo\n")
		}, "fetch and build helix's grammars: hx or the grammar list changed"},
		{"lost library", func(m *effecttest.Machine, g Grammars) {
			m.Files[g.libPath("go")] = nil
			m.Files[stampFile] = []byte("helix 25.07.1 (abc)\ngo\nxml\n")
		}, "fetch and build helix's grammars: xml has no library"},
		{"no hx", func(m *effecttest.Machine, _ Grammars) { delete(m.Files, hxBin) },
			"fetch and build helix's grammars: hx is not installed yet"},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		g := newGrammars(m)
		c.setup(m, g)
		got, err := g.Plan(context.Background())
		switch {
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && len(got) != 0, c.want != "" && (len(got) != 1 || got[0].String() != c.want):
			t.Errorf("%s: changes %v, want %q", c.name, got, c.want)
		}
	}
}

func TestGrammarsFetchBuildAndStamp(t *testing.T) {
	m := effecttest.NewMachine()
	g := newGrammars(m)
	g.UID, g.GID = 1000, 1001
	m.Results["env HOME=/root hx --grammar fetch"] = effecttest.Result{}
	m.Results["env HOME=/root hx --grammar build"] = effecttest.Result{}
	changes, err := g.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"run env HOME=/root hx --grammar fetch", "run env HOME=/root hx --grammar build"}
	if got := m.Calls[len(m.Calls)-3 : len(m.Calls)-1]; !slices.Equal(got, want) {
		t.Errorf("calls %q", got)
	}
	if string(m.Files[stampFile]) != "helix 25.07.1 (abc)\ngo\nxml\n" || m.Owners[stampFile] != [2]int{1000, 1001} {
		t.Errorf("stamp %q owner %v", m.Files[stampFile], m.Owners[stampFile])
	}
}
