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
