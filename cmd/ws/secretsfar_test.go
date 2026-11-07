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
	m, id := editFixture(t, "[env]\nOLD = \"x\"\nA = \"y\"\n")
	for _, a := range [][]string{{"unset", "env", "OLD"}} {
		if status, _, errs := runFarSecrets(m, nil, "dummy-value\n", a...); status != 0 {
			t.Fatalf("%q: status %d, stderr %q", a, status, errs)
		}
	}
	s, err := secrets.Load(m, serverSecrets, id)
	if err != nil || len(s.Env) != 1 {
		t.Errorf("secrets = %+v, %v; want OLD dropped", s, err)
	}
	status, out, _ := runFarSecrets(m, nil, "", "keys")
	if status != 0 || out != "env A\n" {
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
