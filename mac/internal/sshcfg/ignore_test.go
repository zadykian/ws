package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/mac/internal/sshcfg"
)

func TestIgnoreUnknown(t *testing.T) {
	text := "IgnoreUnknown UseKeychain, Foo # c\nHost a\n  ignoreunknown=\"bar\" foo\n"
	if got, want := sshcfg.IgnoreUnknown(text, "UseKeychain"), "UseKeychain,Foo,bar"; got != want {
		t.Errorf("IgnoreUnknown = %q, want %q", got, want)
	}
	if got := sshcfg.IgnoreUnknown("", "UseKeychain"); got != "UseKeychain" {
		t.Errorf("IgnoreUnknown of nothing = %q", got)
	}
}
