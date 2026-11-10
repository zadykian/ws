package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

// tilde writes a path under the home as ~/, as mac.conf and config.d/ws name it.
func (k *MacKey) tilde(path string) string { return "~/" + strings.TrimPrefix(path, k.Home+"/") }

// open opens Secretive's app, or says to by hand.
func (k *MacKey) open(ctx context.Context) {
	if _, err := k.M.Run(ctx, "open", "-a", "Secretive"); err != nil {
		k.Out.Say("%s: warning: open cannot find Secretive: open it by hand", macKey)
	}
}

// secretiveKey has the user make ws's key in Secretive's app, installed where missing, at its
// default protection, which asks for Touch ID at each use. It writes the key's .pub and the agent
// into mac.conf, as git's key would stop the default, and the agent must list it.
func (k *MacKey) secretiveKey(ctx context.Context) error {
	if !k.app() {
		if err := k.M.Pass(ctx, false, "brew", "install", "--cask", "secretive"); err != nil {
			return fmt.Errorf("%s: brew install --cask secretive failed; install Secretive, or answer no for "+
				"ssh-keygen's key", macKey)
		}
	}
	k.open(ctx)
	text, c, err := loadConf(k.file())
	if err != nil {
		return err
	}
	got, err := k.pick(ctx, sshcfg.Expand(c.GitSigningPubKey, k.Home), "a P-256 key for ws's login, at its default protection", "")
	if err != nil {
		return err
	}
	if !hasLine(text, "WS_IDENTITY_PUB_KEY") {
		lines := line("WS_IDENTITY_PUB_KEY", k.tilde(got)) + "\n" + line("WS_IDENTITY_AGENT", k.tilde(k.secretive())+"/socket.ssh") + "\n"
		if err := refused(k.file().add(ctx, lines)); err != nil {
			return err
		}
	}
	if _, c, err = loadConf(k.file()); err != nil {
		return err
	}
	id, err := setup.PickKey(ctx, k.M, c, k.Home)
	if err != nil {
		return err
	}
	if !k.listed(ctx, id.Text, sshcfg.Expand(id.Agent, k.Home)) {
		return fmt.Errorf("%s: Secretive's agent lists no key of %s: finish Secretive's setup, then run the wizard "+
			"again", macKey, sshcfg.Expand(id.Pub, k.Home))
	}
	return nil
}
