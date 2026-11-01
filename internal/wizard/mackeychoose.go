package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

// choose makes ws's key where it has none: in Secretive, where its app is there or the terminal's
// answer is other than no, else in the file of c.pub, named in mac.conf. With no terminal, it stops,
// naming the key that answers.
func (k *MacKey) choose(ctx context.Context, c chooseKey) error {
	if c.app {
		return k.secretiveKey(ctx) // git's key is a change of the plan then
	}
	if !k.Tty {
		return fmt.Errorf("%s: no terminal to ask whether Secretive keeps the Mac's key: set "+
			"WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for ssh-keygen's, or run the wizard at a terminal", macKey)
	}
	answer, err := k.P.Ask(ctx, effect.Question{Text: "Keep the Mac's ssh key in Secretive, in the Secure Enclave? [Y/n]"})
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: the terminal ended before the answer", macKey)
	}
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(answer), "n") {
		if err := k.secretiveKey(ctx); err != nil {
			return err
		}
		return k.git(ctx, true)
	}
	if err := k.make(ctx, strings.TrimSuffix(c.pub, ".pub")); err != nil {
		return err
	}
	return refused(k.file().add(ctx, line("WS_IDENTITY", k.tilde(c.pub))+"\n"))
}

// git has the user make git's signing key in Secretive's app, beside ws's, at Notify, so that git
// signs with no prompt while the Mac is unlocked (#109). It writes its .pub's path into mac.conf as
// WS_GIT_SIGNING_KEY. Where only is set, it does so only for a git that has a name and no line.
func (k *MacKey) git(ctx context.Context, only bool) error {
	text, c, err := loadConf(k.file())
	if err != nil || (only && (c.GitName == "" || hasLine(text, "WS_GIT_SIGNING_KEY"))) {
		return err
	}
	id, err := setup.PickKey(ctx, k.M, c, k.Home)
	if err != nil {
		return err
	}
	k.open(ctx)
	got, err := k.pick(ctx, sshcfg.Expand(id.Pub, k.Home), "a second P-256 key, for git's signing, at Notify",
		"An empty WS_GIT_SIGNING_KEY= in mac.conf keeps ws mac apply's file key for git at the next run")
	if err != nil {
		return err
	}
	return refused(k.file().add(ctx, line("WS_GIT_SIGNING_KEY", k.tilde(got))+"\n"))
}
