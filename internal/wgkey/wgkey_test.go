package wgkey

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// The pair is Alice's from RFC 7748, section 6.1, in base64.
const (
	private = "dwdtCnMYpX08FsFyUbJmRd9ML4frwJkqsXf7pR25LCo="
	public  = "hSDwCYkwp1R0i33ctD73Wg2/Og0mOBr066SpjqqbTmo="
)

func TestGenerate(t *testing.T) {
	random := bytes.NewReader(bytes.Repeat([]byte{0xff}, 32))
	got, err := Generate(random)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(got)
	if raw[0] != 0xf8 || raw[31] != 0x7f {
		t.Errorf("clamped ends = %#x, %#x; want 0xf8, 0x7f", raw[0], raw[31])
	}
	low, _ := Generate(bytes.NewReader(make([]byte, 32)))
	raw, _ = base64.StdEncoding.DecodeString(low)
	if raw[31] != 0x40 {
		t.Errorf("clamped last byte = %#x, want 0x40", raw[31])
	}
}

func TestGenerateShortRandom(t *testing.T) {
	if _, err := Generate(bytes.NewReader([]byte{1, 2})); err == nil {
		t.Error("a short source of random made a key")
	}
}

func TestPublic(t *testing.T) {
	got, err := Public(private)
	if err != nil {
		t.Fatal(err)
	}
	if got != public {
		t.Errorf("Public = %s, want %s", got, public)
	}
	if _, err := Public("short"); err == nil {
		t.Error("Public took a short key")
	}
}

func TestHex(t *testing.T) {
	got, err := Hex(public)
	want := "8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a"
	if err != nil || got != want {
		t.Errorf("Hex = %q, %v; want %q", got, err, want)
	}
	if _, err := Hex(public[:40]); err == nil {
		t.Error("Hex took a key of the wrong size")
	}
}
