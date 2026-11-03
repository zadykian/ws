package keychain

import (
	"context"
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	find  = "security find-generic-password -s ws-secrets"
	notIn = "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain."
)

func TestInitAddsTheItemByStdin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[find] = effecttest.Result{Err: errors.New(notIn)}
	m.Results["security -i"] = effecttest.Result{}
	made, err := Keychain{M: m, User: "me"}.Init(context.Background())
	if err != nil || !made {
		t.Fatalf("Init = %v, %v; want true, nil", made, err)
	}
	if len(m.Inputs) != 1 || !strings.HasPrefix(m.Inputs[0], "add-generic-password -s ws-secrets -a me -w AGE-SECRET-KEY-1") {
		t.Fatalf("stdin = %q, want the add command with a key", m.Inputs)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "AGE-SECRET-KEY") {
			t.Errorf("call %q holds the key", c)
		}
	}
}

func TestInitKeepsAnItem(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[find] = effecttest.Result{}
	made, err := Keychain{M: m, User: "me"}.Init(context.Background())
	if err != nil || made {
		t.Fatalf("Init = %v, %v; want false, nil", made, err)
	}
	if len(m.Inputs) != 0 {
		t.Errorf("stdin = %q, want no command", m.Inputs)
	}
}

func TestInitRefusesAUserThatNeedsQuoting(t *testing.T) {
	for _, u := range []string{"", "a b", "a;b"} {
		if _, err := (Keychain{M: effecttest.NewMachine(), User: u}).Init(context.Background()); err == nil {
			t.Errorf("Init(%q) = nil, want an error", u)
		}
	}
}

func TestInitScrubsTheKeyFromAnError(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[find] = effecttest.Result{Err: errors.New(notIn)}
	m.Results["security -i"] = effecttest.Result{Err: errors.New("exit 1")}
	_, err := Keychain{M: m, User: "me"}.Init(context.Background())
	if err == nil || strings.Contains(err.Error(), "AGE-SECRET-KEY") {
		t.Errorf("err = %v, want one with no key", err)
	}
}

func TestIdentityReadsTheItem(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	m.Results[find+" -w"] = effecttest.Result{Out: []byte(id.String() + "\n")}
	got, err := Keychain{M: m}.Identity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.(*age.X25519Identity).Recipient().String() != id.Recipient().String() {
		t.Error("the identity differs")
	}
}

func TestIdentityErrors(t *testing.T) {
	tests := []struct {
		name string
		res  effecttest.Result
		want string
	}{
		{"missing", effecttest.Result{Err: errors.New(notIn)}, "ws mac secrets init"},
		{"locked", effecttest.Result{Err: errors.New("exit status 36")}, "can't read the item"},
		{"not a key", effecttest.Result{Out: []byte("MY_TOKEN\n")}, "holds no age identity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[find+" -w"] = tt.res
			_, err := Keychain{M: m}.Identity(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) || strings.Contains(err.Error(), "MY_TOKEN") {
				t.Errorf("err = %v, want %q and no value", err, tt.want)
			}
		})
	}
}
