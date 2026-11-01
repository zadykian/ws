package wizard

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/mac/internal/step"
)

// fileKey checks the key in a file: it plans to make it where neither it nor its .pub is there,
// and fails where one is missing or the two differ.
func (k *MacKey) fileKey(ctx context.Context, key string) ([]step.Change, error) {
	switch {
	case !k.exists(key) && !k.exists(key+".pub"):
		return []step.Change{makeKey{key}}, nil
	case !k.exists(key):
		return nil, fmt.Errorf("%s: %s.pub is there without its private key, %s: remove it, or name the agent that "+
			"holds the key in mac.conf's WS_IDENTITY_AGENT", macKey, key, key)
	case !k.same(ctx, key):
		return nil, fmt.Errorf("%s: %s.pub is not %s's public key: ssh-keygen -y -f %s >%s.pub writes it",
			macKey, key, key, key, key)
	}
	return nil, nil
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
