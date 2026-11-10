package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/effect"
)

// choose makes ws's key where it has none. The key goes in Secretive where its app is there or the
// terminal's answer is other than no. Else it goes in the file of c.pub, named in mac.conf. With no
// terminal, it stops, naming the key that answers.
func (k *MacKey) choose(ctx context.Context, c chooseKey) error {
	if c.app {
		return k.secretiveKey(ctx)
	}
	if !k.Tty {
		return fmt.Errorf("%s: no terminal to ask whether Secretive keeps the Mac's key: set "+
			"WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub in mac.conf for ssh-keygen's, or run the wizard at a terminal", macKey)
	}
	answer, err := k.P.Ask(ctx, effect.Question{Text: "Keep the Mac's ssh key in Secretive, in the Secure Enclave? [Y/n]"})
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: the terminal ended before the answer", macKey)
	}
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(answer), "n") {
		return k.secretiveKey(ctx)
	}
	if err := k.make(ctx, strings.TrimSuffix(c.pub, ".pub")); err != nil {
		return err
	}
	return refused(k.file().add(ctx, line("WS_IDENTITY_PUB_KEY", k.tilde(c.pub))+"\n"))
}
