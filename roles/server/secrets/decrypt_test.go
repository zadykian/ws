package secrets

import (
	"testing"

	"filippo.io/age"
)

// encrypt returns plain as an armored age file for the recipient.
func encrypt(tb testing.TB, plain string, to age.Recipient) []byte {
	out, err := Encrypt([]byte(plain), to)
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

func TestDecryptRoundTrip(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(encrypt(t, "k: v\n", id.Recipient()), id)
	if err != nil || string(got) != "k: v\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestDecryptRefusesAnotherIdentityAndPlainText(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	other, _ := age.GenerateX25519Identity()
	if _, err := Decrypt(encrypt(t, "secret", id.Recipient()), other); err == nil {
		t.Error("another identity decrypted")
	}
	if _, err := Decrypt([]byte("k: v\n"), id); err == nil {
		t.Error("plain text decrypted")
	}
}
