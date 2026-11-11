package wizard

import (
	"context"
	"fmt"
)

// fileKey checks the key in a file, which the stage never makes: it fails where the key or its .pub
// is missing, or the two differ.
func (k *MacKey) fileKey(ctx context.Context, key string) error {
	switch {
	case !k.exists(key) && !k.exists(key+".pub"):
		return fmt.Errorf("%s: %s, named in mac.conf's WS_IDENTITY_PUB_KEY, is missing: make the key, or drop the "+
			"line for the wizard to ask for one of Secretive's", macKey, key+".pub")
	case !k.exists(key):
		return fmt.Errorf("%s: %s.pub is there without its private key, %s: remove it, or name the agent that "+
			"holds the key in mac.conf's WS_IDENTITY_AGENT", macKey, key, key)
	case !k.same(ctx, key):
		return fmt.Errorf("%s: %s.pub is not %s's public key: ssh-keygen -y -f %s >%s.pub writes it",
			macKey, key, key, key, key)
	}
	return nil
}

func (k *MacKey) exists(path string) bool {
	_, err := k.M.Stat(path)
	return err == nil
}

// app reports whether Secretive's app is in /Applications, where its cask puts it, or in
// ~/Applications.
func (k *MacKey) app() bool {
	return k.exists("/Applications/Secretive.app") || k.exists(k.Home+"/Applications/Secretive.app")
}
