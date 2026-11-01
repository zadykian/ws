package wizard

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestPkgSigner(t *testing.T) {
	good := "Package \"Homebrew.pkg\":\n   Status: signed by a developer certificate issued by Apple for distribution\n" +
		"   Notarization: trusted by the Apple notary service\n   Certificate Chain:\n" +
		"    1. " + brewSigner + "\n       Expires: 2030\n    2. Developer ID Certification Authority\n"
	for name, tc := range map[string]struct{ out, want string }{
		"issued by Apple": {good, brewSigner},
		"no status":       {"   1. Mallory\n", ""},
		"before status": {"    1. Mallory\n   Status: signed by a developer certificate issued by Apple for distribution\n" +
			"    1. " + brewSigner + "\n", brewSigner},
		"unsigned": {"Package \"x.pkg\":\n   Status: no signature\n", ""},
		"empty":    {"", ""},
	} {
		if got := pkgSigner(tc.out); got != tc.want {
			t.Errorf("%s: pkgSigner = %q, want %q", name, got, tc.want)
		}
	}
}

func TestBrewSHA256IsHex(t *testing.T) {
	if b, err := hex.DecodeString(brewSHA256); err != nil || len(b) != sha256.Size {
		t.Errorf("brewSHA256 = %q, want %d hex bytes", brewSHA256, sha256.Size)
	}
}
