package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

// runFarSecrets runs ws mac secrets ARGS on a Mac whose ws is m, or where ssh fails with err.
func runFarSecrets(m *effecttest.Machine, err error, stdin string, args ...string) (int, string, string) {
	p, mac := macTerminal()
	mac.Files["/p12"] = []byte{0, 1, 2}
	p.machine = wsSecrets{farWs: newFarWs(mac, ""), ws: m, err: err}
	return runMacSecretsOn(p, stdin, args...)
}

// runMacSecretsOn runs ws mac secrets ARGS, with stdin.
func runMacSecretsOn(p process, stdin string, args ...string) (int, string, string) {
	var out, errs bytes.Buffer
	root := newRoot(p, group("mac", "the Mac's commands", newMacSecretsCmd(p)))
	status := execute(context.Background(), root, append([]string{"mac", "secrets"}, args...), strings.NewReader(stdin), &out, &errs)
	return status, out.String(), errs.String()
}

func TestMacSecretsEditWsFile(t *testing.T) {
	m, id := editFixture(t, "[env]\nOLD = \"x\"\n")
	for _, a := range [][]string{{"set", "env", "A"}, {"set", "git", "https://h.example", "--username", "bob"},
		{"set", "file", "vpn_p12", "/p12"}, {"unset", "env", "OLD"}} {
		if status, _, errs := runFarSecrets(m, nil, "dummy-value\n", a...); status != 0 {
			t.Fatalf("%q: status %d, stderr %q", a, status, errs)
		}
	}
	s, err := secrets.Load(m, serverSecrets, id)
	if g := s.Git["https://h.example"]; err != nil || s.Env["A"] != "dummy-value" || len(s.Env) != 1 ||
		g.Token != "dummy-value" || g.Username != "bob" || len(s.File.VPNP12) != 3 {
		t.Errorf("secrets = %+v, %v; want what was set", s, err)
	}
	status, out, _ := runFarSecrets(m, nil, "", "keys")
	if status != 0 || out != "env A\nfile vpn_p12\ngit https://h.example\n" {
		t.Errorf("keys = %d %q", status, out)
	}
}

func TestMacSecretsSayWhyTheyStop(t *testing.T) {
	ssh := errors.New("[ssh]: exit status 255: \nssh: connect to host ws: No route to host")
	for err, want := range map[error]string{ssh: "ws is out of reach", nil: "run ws server secrets init"} {
		status, _, errs := runFarSecrets(effecttest.NewMachine(), err, "", "unset", "env", "A")
		if status != 1 || !strings.Contains(errs, want) {
			t.Errorf("status %d, stderr %q; want 1 naming %q", status, errs, want)
		}
	}
}

// With no way to ws, a command says so before it reads a value, and a value never shows.
func TestMacSecretsSetAsksNoValueWithWsOutOfReach(t *testing.T) {
	p, m := macTerminal()
	if status, _, errs := runMacSecretsOn(p, "dummy-value\n", "set", "env", "A"); status != 1 ||
		!strings.Contains(errs, "no direct way") || len(m.Inputs) != 0 {
		t.Errorf("status %d, stderr %q; want 1 naming the way", status, errs)
	}
	status, _, errs := runFarSecrets(effecttest.NewMachine(), nil, "dummy-value\n", "set", "env", "A")
	if status != 1 || !strings.Contains(errs, "run ws server secrets init") || strings.Contains(errs, "dummy-value") {
		t.Errorf("status %d, stderr %q; want 1 naming init, and no value", status, errs)
	}
}
