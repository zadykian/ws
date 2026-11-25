package ikeward

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func newProfile(m *effecttest.Machine) profile {
	return profile{m: m, name: "/root/vpn/vpnclient", files: Profile{P12: []byte("p12"), Sswan: []byte("sswan")}}
}

// lookupOf looks a variable up in env, as the run's env would.
func lookupOf(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

func TestProfileFromUnset(t *testing.T) {
	p, set, err := ProfileFrom(lookupOf(map[string]string{PasswordVar: "hunter2"}))
	if set || err != nil || len(p.P12)+len(p.Sswan)+len(p.Password) != 0 {
		t.Errorf("ProfileFrom = %v, %v, %v; want nothing, unset", p, set, err)
	}
}

func TestProfileFromDecodes(t *testing.T) {
	// base64 without -w0 wraps at 76 columns; the decode takes both forms.
	p12 := []byte(strings.Repeat("\x00\xffkey", 30))
	enc := base64.StdEncoding.EncodeToString(p12)
	for _, in := range []string{enc, enc[:76] + "\n" + enc[76:] + "\n"} {
		p, set, err := ProfileFrom(lookupOf(map[string]string{P12Var: in, SswanVar: "{\"x\": 1}", PasswordVar: "hunter2"}))
		if !set || err != nil || string(p.P12) != string(p12) || string(p.Sswan) != `{"x": 1}` || p.Password != "hunter2" {
			t.Errorf("ProfileFrom = %v, %v; want the decoded profile", set, err)
		}
	}
	p, set, err := ProfileFrom(lookupOf(map[string]string{P12Var: "", SswanVar: ""}))
	if !set || err != nil || len(p.P12)+len(p.Sswan) != 0 {
		t.Errorf("ProfileFrom of empty variables = %v, %v; want set and empty", set, err)
	}
}

func TestProfileFromBadBase64NamesTheVariable(t *testing.T) {
	_, set, err := ProfileFrom(lookupOf(map[string]string{P12Var: "not-base64-secret!", SswanVar: "s"}))
	if !set || err == nil || !strings.Contains(err.Error(), P12Var) {
		t.Fatalf("ProfileFrom = %v, %v; want an error naming %s", set, err, P12Var)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error %q holds the value", err)
	}
}

func TestWithWritesTheFilesUnderTheirNamesAndRemovesThem(t *testing.T) {
	m := effecttest.NewMachine()
	var seen string
	var mode fs.FileMode
	err := newProfile(m).with(t.Context(), func() error {
		seen = string(m.Files[runDir+"/vpnclient.p12"]) + string(m.Files[runDir+"/vpnclient.sswan"])
		mode = m.Dirs[runDir]
		return nil
	})
	if err != nil || seen != "p12sswan" {
		t.Errorf("with = %v, files %q; want the two files while f runs", err, seen)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+runDir) {
		t.Errorf("calls = %q, lack the removal", m.Calls)
	}
	if mode != 0o700 {
		t.Errorf("mode of %s = %o, want 700", runDir, mode)
	}
	if _, left := m.Dirs[runDir]; left {
		t.Errorf("%s was left", runDir)
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
