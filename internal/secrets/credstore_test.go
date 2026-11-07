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
	got, err := Credstore{M: m}.Identity(context.Background())
	if err != nil || got.(*age.X25519Identity).String() != id.String() {
		t.Errorf("Identity = %v; want the credential's identity", err)
	}
}

func TestCredstoreWithoutTheCredential(t *testing.T) {
	_, err := Credstore{M: effecttest.NewMachine()}.Identity(context.Background())
	if !errors.Is(err, ErrNoCredential) || !strings.Contains(err.Error(), "ws server secrets init") {
		t.Errorf("err = %v, want ErrNoCredential naming init", err)
	}
}

func TestCredstoreErrorShowsNoKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[CredstorePath] = []byte("sealed")
	m.Results[credDecrypt] = effecttest.Result{Out: []byte("AGE-SECRET-KEY-nope\n")}
	if _, err := (Credstore{M: m}).Identity(context.Background()); err == nil || strings.Contains(err.Error(), "SECRET-KEY") {
		t.Errorf("Identity: err = %v, want one with no key", err)
	}
}
