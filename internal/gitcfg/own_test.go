package gitcfg_test

import (
	"testing"

	"github.com/zadykian/ws/internal/gitcfg"
)

func TestOwnListsEachSettingOnce(t *testing.T) {
	names := []string{"core.editor", "user.name", "gpg.format", "user.name", "user.signingkey", "gpg.ssh.program"}
	if got, want := gitcfg.Own(names), "user.name, gpg.format, user.signingkey, gpg.ssh.program"; got != want {
		t.Errorf("Own = %q, want %q", got, want)
	}
	if got := gitcfg.Own(nil); got != "" {
		t.Errorf("Own(nil) = %q", got)
	}
}
