package wizard

import (
	"fmt"
	"testing"
)

func TestMacKeyChanges(t *testing.T) {
	for want, c := range map[string]fmt.Stringer{
		"Mac key: would make /h/k with ssh-keygen -t ed25519 -a 100, then run ssh-add --apple-use-keychain /h/k":                                              makeKey{"/h/k"},
		"Mac key: would open Secretive, and wait for ws's key made there":                                                                                     chooseKey{pub: "/h/k.pub", app: true},
		"Mac key: would ask whether Secretive keeps the Mac's key: yes installs its cask, no makes /h/k with ssh-keygen, and names it in WS_IDENTITY_PUB_KEY": chooseKey{pub: "/h/k.pub"},
		"Mac key: would open Secretive, and wait for git's signing key made there":                                                                            gitKey{},
	} {
		if got := c.String(); got != want {
			t.Errorf("String = %q, want %q", got, want)
		}
	}
}
