package wizard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"testing"

	agelib "filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

const macFile = "/h/.config/ws/secrets.age"

// withSecrets gives the stage the Mac's file and a Keychain that holds the identity, and ws the
// remote script's answers for the secrets.
func withSecrets(t *testing.T, status string) (*WsRun, *effecttest.Machine, string) {
	t.Helper()
	r, m := newRun("lan", status, false, false)
	id, _ := agelib.GenerateX25519Identity()
	r.Secrets, r.User = macFile, "me"
	m.Files[macFile] = []byte("armored bytes")
	m.Results["security find-generic-password -s ws-secrets -w"] = effecttest.Result{Out: []byte(id.String() + "\n")}
	m.Results[remoteLine(r.Remote, "secrets", "stage")] = effecttest.Result{
		Out: []byte("/var/lib/ws-wizard/secrets.age /run/ws-wizard/identity\n")}
	m.Results[remoteLine(r.Remote, "run")] = effecttest.Result{}
	for _, path := range []string{"/var/lib/ws-wizard/secrets.age", "/run/ws-wizard/identity"} {
		m.Results[sendLine(r, path)] = effecttest.Result{}
	}
	return r, m, id.String()
}

func sendLine(r *WsRun, path string) string {
	return "ssh " + strings.Join(r.Host.SSH.argv([]string{"-o", "BatchMode=yes", "ws-ssh", "umask 077 && cat >" + path}), " ")
}

func sum(b string) string {
	h := sha256.Sum256([]byte(b))
	return hex.EncodeToString(h[:])
}

func TestWsRunSendsTheFileAndTheIdentityOnStdin(t *testing.T) {
	r, m, key := withSecrets(t, "run - -\nactive -\n")
	p, err := r.Plan(context.Background())
	if err != nil || !strings.Contains(p.Changes[0].String(), "after sending the secrets file and the identity") {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if want := []string{"armored bytes", key + "\n"}; !slices.Equal(m.Inputs[len(m.Inputs)-3:len(m.Inputs)-1], want) {
		t.Errorf("inputs = %q; want the file, then the identity", m.Inputs)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, key) {
			t.Errorf("the identity is in the call %q", c)
		}
	}
}

func TestWsRunSendsAgainForEachRun(t *testing.T) {
	r, m, _ := withSecrets(t, "run v1.2.3 0 on -\nactive -\noutbound on\n")
	r.Host.Conf = "/h/cloud.conf"
	m.Files["/h/cloud.conf"] = []byte("WS_TARGET=cloud\n" + extra["cloud"])
	m.Results[remoteLine(r.Remote, "outbound")] = effecttest.Result{}
	p, err := r.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	stages := 0
	for _, c := range m.Calls {
		if strings.HasSuffix(c, "-- secrets stage") {
			stages++
		}
	}
	if stages != 2 {
		t.Errorf("the secrets went %d times; want once for each of the two runs", stages)
	}
}

func TestWsRunWithoutTheFileDropsWsFile(t *testing.T) {
	r, m, _ := withSecrets(t, "run - -\nactive -\n")
	delete(m.Files, macFile)
	p, _ := r.Plan(context.Background())
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run "+remoteLine(r.Remote, "secrets", "drop")) ||
		slices.Contains(m.Calls, "run "+remoteLine(r.Remote, "secrets", "stage")) {
		t.Errorf("calls = %q; want a drop and no stage", m.Calls)
	}
}

func TestWsRunIsDoneAtTheFilesSum(t *testing.T) {
	r, m, _ := withSecrets(t, "run v1.2.3 0 on "+sum("armored bytes")+"\nactive -\n")
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("same file: %v, %v; want none", p, err)
	}
	m.Files[macFile] = []byte("another file")
	r.Remote.Forget()
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Errorf("new file: %v, %v; want a run", p, err)
	}
	delete(m.Files, macFile)
	if p, _ := r.Plan(context.Background()); len(p.Changes) != 1 {
		t.Error("the file gone, the stamp naming one: want a run")
	}
}

func TestWsRunNamesInitWhereTheKeychainLacksTheItem(t *testing.T) {
	r, m, _ := withSecrets(t, "run - -\nactive -\n")
	m.Results["security find-generic-password -s ws-secrets -w"] = effecttest.Result{
		Err: errors.New("exit status 44")}
	p, _ := r.Plan(context.Background())
	if err := r.Apply(context.Background(), p); err == nil || !strings.Contains(err.Error(), "ws mac secrets init") {
		t.Errorf("Apply = %v; want the name of init", err)
	}
	if slices.Contains(m.Calls, "run "+remoteLine(r.Remote, "secrets", "stage")) {
		t.Error("something was sent without the identity")
	}
}

func TestWsRunDropsWhatWsHoldsWhereARunFails(t *testing.T) {
	for name, tc := range map[string]struct {
		identity bool
		want     []string
	}{
		"run fails":          {false, []string{"secrets", "drop", "identity"}},
		"the identity fails": {true, []string{"secrets", "drop"}},
	} {
		r, m, _ := withSecrets(t, "run - -\nactive -\n")
		fail := remoteLine(r.Remote, "run")
		if tc.identity {
			fail = sendLine(r, "/run/ws-wizard/identity")
		}
		m.Results[fail] = effecttest.Result{Err: errors.New("exit status 255")}
		m.Results[remoteLine(r.Remote, tc.want...)] = effecttest.Result{}
		p, _ := r.Plan(context.Background())
		if err := r.Apply(context.Background(), p); err == nil {
			t.Fatalf("%s: Apply = nil, want the failure", name)
		}
		if !slices.Contains(m.Calls, "run "+remoteLine(r.Remote, tc.want...)) {
			t.Errorf("%s: calls = %q; want %v", name, m.Calls, tc.want)
		}
	}
}
