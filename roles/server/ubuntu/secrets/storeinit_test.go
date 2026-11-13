package secrets

import (
	"context"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestStoreInitMakesBoth(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Results[credEncrypt] = effecttest.Result{Out: []byte("sealed")}
	m.Results[credDecrypt] = effecttest.Result{Out: []byte(id.String() + "\n")}
	found, err := Store{M: m}.Init(context.Background())
	if err != nil || found != (Found{}) {
		t.Fatalf("Init = %+v, %v; want nothing found", found, err)
	}
	if _, err := Load(m, FilePath, id); err != nil || m.Modes[CredstorePath] != 0o600 || m.Modes[FilePath] != 0o600 {
		t.Errorf("Load = %v, modes %o and %o; want an empty file that the identity opens, 600 for both", err,
			m.Modes[CredstorePath], m.Modes[FilePath])
	}
}

func TestStoreInitKeepsWhatIsThere(t *testing.T) {
	m, _ := storeFixture(t, "[env]\nA = \"dummy\"\n")
	before := string(m.Files[FilePath])
	found, err := Store{M: m}.Init(context.Background())
	if err != nil || found != (Found{Credential: true, File: true}) || string(m.Files[FilePath]) != before {
		t.Errorf("Init = %+v, %v; want both found and the file as it was", found, err)
	}
}

func TestStoreInitMakesTheFileBesideTheCredential(t *testing.T) {
	m, id := storeFixture(t, "")
	found, err := Store{M: m}.Init(context.Background())
	if err != nil || found != (Found{Credential: true}) {
		t.Fatalf("Init = %+v, %v; want the credential found", found, err)
	}
	if _, err := Load(m, FilePath, id); err != nil {
		t.Errorf("Load = %v; want an empty file that the identity opens", err)
	}
}

func TestStoreInitRefusesAFileAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[FilePath] = []byte("kept")
	_, err := Store{M: m}.Init(context.Background())
	if err == nil || !strings.Contains(err.Error(), "remove the file, then run init again") ||
		string(m.Files[FilePath]) != "kept" || len(m.Inputs) != 0 {
		t.Errorf("err = %v; want the refusal, with the file kept and no identity made", err)
	}
}
