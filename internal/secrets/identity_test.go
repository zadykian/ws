package secrets

import (
	"io"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestReadIdentityFromAPipe(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	in := "# created: now\n" + id.String() + "\n"
	got, err := ReadIdentity(strings.NewReader(in), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(encrypt(t, "x", id.Recipient()), got)
	if err != nil || string(plain) != "x" {
		t.Errorf("got %q, %v", plain, err)
	}
}

func TestReadIdentityRefuses(t *testing.T) {
	a, _ := age.GenerateX25519Identity()
	b, _ := age.GenerateX25519Identity()
	for name, in := range map[string]string{
		"empty": "", "garbage": "AGE-SECRET-KEY-nope\n", "two": a.String() + "\n" + b.String() + "\n",
	} {
		_, err := ReadIdentity(strings.NewReader(in), io.Discard)
		if err == nil {
			t.Errorf("%s: no error", name)
		} else if strings.Contains(err.Error(), "SECRET-KEY") {
			t.Errorf("%s: the error shows the key: %v", name, err)
		}
	}
}
