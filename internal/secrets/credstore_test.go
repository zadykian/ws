package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

const credDecrypt = "systemd-creds decrypt --name=ws-secrets " + CredstorePath + " -"

func TestCredstoreReadsTheIdentity(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Files[CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte(id.String() + "\n")}
	got, err := credstore{M: m}.Identity(context.Background())
	if err != nil || got.(*age.X25519Identity).String() != id.String() {
		t.Errorf("Identity = %v; want the credential's identity", err)
	}
}

func TestCredstoreWithoutTheCredential(t *testing.T) {
	_, err := credstore{M: effecttest.NewMachine()}.Identity(context.Background())
	if !errors.Is(err, ErrNoCredential) || !strings.Contains(err.Error(), "ws server secrets init") {
		t.Errorf("err = %v, want ErrNoCredential naming init", err)
	}
}

func TestCredstoreErrorShowsNoKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte("AGE-SECRET-KEY-nope\n")}
	if _, err := (credstore{M: m}).Identity(context.Background()); err == nil || strings.Contains(err.Error(), "SECRET-KEY") {
		t.Errorf("Identity: err = %v, want one with no key", err)
	}
}

const credEncrypt = "systemd-creds encrypt --with-key=host --name=ws-secrets - -" //nolint:gosec // G101: a command line, no credential

func TestCredstoreMakesTheIdentity(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[credEncrypt] = effecttest.Result{Out: []byte("sealed")}
	id, err := credstore{M: m}.Make(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Inputs) != 1 || m.Inputs[0] != id.String()+"\n" {
		t.Error("the key did not go on systemd-creds' stdin")
	}
	if string(m.Files[CredstorePath]) != "sealed" || m.Modes[CredstorePath] != 0o600 ||
		m.Dirs["/etc/credstore.encrypted"] != 0o700 {
		t.Errorf("credential %q, mode %o, directory %o; want sealed, 600, 700",
			m.Files[CredstorePath], m.Modes[CredstorePath], m.Dirs["/etc/credstore.encrypted"])
	}
}

func TestCredstoreFailedMakeKeepsTheCredential(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[credEncrypt] = effecttest.Result{Err: errors.New("exit status 1")}
	m.Files[CredstorePath] = []byte("sealed")
	if _, err := (credstore{M: m}).Make(context.Background()); err == nil || string(m.Files[CredstorePath]) != "sealed" {
		t.Errorf("Make = %v; want an error, and the credential kept", err)
	}
	if err := scrubKey(errors.New("echoed KEY"), "KEY"); err.Error() != "echoed [key]" {
		t.Errorf("scrubKey = %v", err)
	}
}
