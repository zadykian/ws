package wizard

import (
	"fmt"
	"strings"
)

// makeKey is the change that makes ws's key in a file.
type makeKey struct{ path string }

func (c makeKey) String() string {
	return fmt.Sprintf("%s: would make %s with ssh-keygen -t ed25519 -a 100, then run ssh-add --apple-use-keychain %s",
		macKey, c.path, c.path)
}

// chooseKey is the change that makes ws's key where it has none. The key goes in Secretive, or in
// the file at pub where the answer is no. app tells whether Secretive's app is there.
type chooseKey struct {
	pub string
	app bool
}

func (c chooseKey) String() string {
	if c.app {
		return macKey + ": would open Secretive, and wait for ws's key made there"
	}
	return fmt.Sprintf("%s: would ask whether Secretive keeps the Mac's key: yes installs its cask, no makes %s with "+
		"ssh-keygen, and names it in WS_IDENTITY_PUB_KEY", macKey, strings.TrimSuffix(c.pub, ".pub"))
}
