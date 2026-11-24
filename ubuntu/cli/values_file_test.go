package cli

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const valuesFile = "/etc/ws/values.yml"

// textOf is the whole file's text for the values that in gives, so that two values compare key by
// key.
func textOf(t *testing.T, in []byte) string {
	v, err := values.Parse(in)
	if err != nil {
		t.Fatalf("the values %q: %v", in, err)
	}
	text, err := v.YAML()
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestValuesFileWrittenWhereNone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o755, 0o755
	steps := specUnits(t, m, &effecttest.Out{})
	if got, err := planned(t, steps, "file/ws-values"); err != nil || len(got) != 1 {
		t.Fatalf("the plan: %q, %v; want the file made", got, err)
	}
	apply(t, steps, "file/ws-values")
	if mode, owner := m.Modes[valuesFile], m.Owners[valuesFile]; mode != 0o600 || owner != [2]int{} {
		t.Errorf("mode %o, owner %v; want root's at 600", mode, owner)
	}
	if got, want := textOf(t, m.Files[valuesFile]), textOf(t, nil); got != want {
		t.Errorf("the file reads as:\n%s\nwant no file's values:\n%s", got, want)
	}
	if got, err := planned(t, specUnits(t, m, &effecttest.Out{}), "file/ws-values"); err != nil || len(got) != 0 {
		t.Errorf("the next plan: %q, %v; want nothing", got, err)
	}
}

// TestValuesFileLeavesAFileOrALink holds the file as the owner's: the unit reads nothing of it.
func TestValuesFileLeavesAFileOrALink(t *testing.T) {
	for name, place := range map[string]func(*effecttest.Machine){
		"a file": func(m *effecttest.Machine) { m.Files[valuesFile] = []byte("") },
		"a link": func(m *effecttest.Machine) { m.Links[valuesFile] = "/nowhere" },
	} {
		m := effecttest.NewMachine()
		m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o755, 0o755
		place(m)
		if got, err := planned(t, specUnits(t, m, &effecttest.Out{}), "file/ws-values"); err != nil || len(got) != 0 {
			t.Errorf("with %s: %q, %v; want nothing planned", name, got, err)
		}
	}
}
