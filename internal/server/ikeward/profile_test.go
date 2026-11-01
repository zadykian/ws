package ikeward

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/secrets"
)

func newProfile(m *effecttest.Machine) profile {
	return profile{m: m, name: "/root/vpn/vpnclient", files: secrets.Files{P12: []byte("p12"), Sswan: []byte("sswan")}}
}

func TestWithWritesTheFilesUnderTheirNamesAndRemovesThem(t *testing.T) {
	m := effecttest.NewMachine()
	var seen string
	err := newProfile(m).with(t.Context(), func() error {
		seen = string(m.Files[runDir+"/vpnclient.p12"]) + string(m.Files[runDir+"/vpnclient.sswan"])
		return nil
	})
	if err != nil || seen != "p12sswan" {
		t.Errorf("with = %v, files %q; want the two files while f runs", err, seen)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+runDir) {
		t.Errorf("calls = %q, lack the removal", m.Calls)
	}
	if m.Dirs[runDir] != 0o700 {
		t.Errorf("mode of %s = %o, want 700", runDir, m.Dirs[runDir])
	}
}

func TestWithRemovesTheFilesWhereFFailsAndKeepsItsError(t *testing.T) {
	m := effecttest.NewMachine()
	boom := errors.New("boom")
	if err := newProfile(m).with(t.Context(), func() error { return boom }); !errors.Is(err, boom) {
		t.Errorf("error = %v, want boom", err)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+runDir) {
		t.Errorf("calls = %q, lack the removal", m.Calls)
	}
}

func TestWithTakesADirectoryALostRunLeftAndSetsItsMode(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs[runDir] = 0o755
	if err := newProfile(m).with(t.Context(), func() error { return nil }); err != nil {
		t.Error(err)
	}
	if !slices.Contains(m.Calls, "chmod "+runDir+" 700") {
		t.Errorf("calls = %q, lack the chmod", m.Calls)
	}
}
