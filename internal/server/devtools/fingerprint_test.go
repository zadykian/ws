package devtools

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // OpenPGP v4 fingerprints are SHA-1 (RFC 9580)
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func newKey(t *testing.T, v6 bool) *openpgp.Entity {
	cfg := &packet.Config{V6Keys: v6, Algorithm: packet.PubKeyAlgoEdDSA}
	if v6 {
		cfg.Algorithm = packet.PubKeyAlgoEd25519
	}
	e, err := openpgp.NewEntity("test", "", "test@example.org", cfg)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func public(t *testing.T, e *openpgp.Entity) []byte {
	var buf bytes.Buffer
	if err := e.Serialize(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// armored wraps the packets as one armored block.
func armored(data ...[]byte) string {
	return block("", base64.StdEncoding.EncodeToString(slices.Concat(data...)), "")
}

func upper(b []byte) string { return strings.ToUpper(hex.EncodeToString(b)) }

func TestFingerprintsOfV4AndV6Keys(t *testing.T) {
	v4, v6 := newKey(t, false), newKey(t, true)
	got, err := fingerprints(armored(public(t, v4)))
	if err != nil || !slices.Equal(got, []string{upper(v4.PrimaryKey.Fingerprint)}) {
		t.Errorf("v4: got %v, %v", got, err)
	}
	if len(v4.PrimaryKey.Fingerprint) != sha1.Size {
		t.Errorf("v4: a fingerprint of %d octets", len(v4.PrimaryKey.Fingerprint))
	}
	got, err = fingerprints(armored(public(t, v6)))
	if err != nil || !slices.Equal(got, []string{upper(v6.PrimaryKey.Fingerprint)}) {
		t.Errorf("v6: got %v, %v", got, err)
	}
	if len(v6.PrimaryKey.Fingerprint) != sha256.Size {
		t.Errorf("v6: a fingerprint of %d octets", len(v6.PrimaryKey.Fingerprint))
	}
}

// TestFingerprintHashesThePacket checks the fingerprint against RFC 9580's definition, which the
// test computes from the packet's own octets: 0x99, a two-octet length and the body.
func TestFingerprintHashesThePacket(t *testing.T) {
	e := newKey(t, false)
	var buf bytes.Buffer
	if err := e.PrimaryKey.Serialize(&buf); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()[2:]                                            // a new-format header of two octets: the tag, one octet of length
	sum := sha1.Sum(append([]byte{0x99, 0, byte(len(body))}, body...)) //nolint:gosec // as above
	got, err := fingerprints(armored(public(t, e)))
	if err != nil || len(got) != 1 || got[0] != upper(sum[:]) {
		t.Errorf("got %v, %v; want %s", got, err, upper(sum[:]))
	}
}

func TestFingerprintsOfSeveralCertificates(t *testing.T) {
	a, b := newKey(t, false), newKey(t, true)
	got, err := fingerprints(armored(public(t, a), public(t, b)))
	want := []string{upper(a.PrimaryKey.Fingerprint), upper(b.PrimaryKey.Fingerprint)}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %v, %v; want %v", got, err, want)
	}
}
