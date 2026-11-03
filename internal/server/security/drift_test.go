package security

import (
	"errors"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const unshare = "unshare --net sh -c \"$1\" \"$2\" && nft -s list table inet ws sh " + scriptFile + " " + tableFile

func driftFixture() (*effecttest.Machine, drift) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files[scriptFile], m.Files[tableFile] = []byte("#!/bin/sh\n"), []byte("table inet ws {}\n")
	return m, drift{m, &effecttest.Out{}}
}

func TestDriftComparesTheKernelsTableWithTheFiles(t *testing.T) {
	for name, c := range map[string]struct {
		kernel effecttest.Result
		want   int
	}{
		"same":     {effecttest.Result{Out: []byte("table\n")}, 0},
		"other":    {effecttest.Result{Out: []byte("other\n")}, 1},
		"no table": {effecttest.Result{Err: errors.New("no such table")}, 1},
	} {
		m, d := driftFixture()
		m.Results[unshare] = effecttest.Result{Out: []byte("table\n")}
		m.Results["nft -s list table inet ws"] = c.kernel
		got, err := d.Plan(t.Context())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: changes = %v, %v; want %d", name, got, err, c.want)
		}
	}
}

func TestDriftTakesAFileThatFailsToLoadAsAReloadDue(t *testing.T) {
	m, d := driftFixture()
	m.Results[unshare] = effecttest.Result{Err: errors.New("syntax error")}
	if got, err := d.Plan(t.Context()); err != nil || len(got) != 1 {
		t.Errorf("changes = %v, %v; want one reload", got, err)
	}
}

func TestDriftWaitsForTheFilesAndForSystemd(t *testing.T) {
	m, d := driftFixture()
	delete(m.Files, tableFile)
	if got, err := d.Plan(t.Context()); err != nil || got != nil {
		t.Errorf("no file: changes = %v, %v; want none", got, err)
	}
	delete(m.Dirs, "/run/systemd/system")
	if got, err := d.Plan(t.Context()); err != nil || got != nil {
		t.Errorf("no systemd: changes = %v, %v; want none", got, err)
	}
}
