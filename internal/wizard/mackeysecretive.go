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

// made has the user make what in Secretive's app, installed where missing, and returns the key
// once the agent lists it beside the keys of before.
func (k *MacKey) made(ctx context.Context, what string, before []sshcfg.AgentKey) (sshcfg.AgentKey, error) {
	if !k.app() {
		if err := k.M.Pass(ctx, false, "brew", "install", "--cask", "secretive"); err != nil {
			return sshcfg.AgentKey{}, fmt.Errorf("%s: brew install --cask secretive failed: install Secretive, "+
				"then run the wizard again", macKey)
		}
	}
	k.open(ctx)
	// The agent's first answer holds the keys there before the wait. Where the agent wasn't there at
	// the question, as before Secretive starts, a key kept from an earlier install may be among them.
	// No answer named it, so the wizard stops, for a run that offers it.
	now, err := k.answer(ctx)
	if err != nil {
		return sshcfg.AgentKey{}, err
	}
	if key := unlisted(now, before); key.Name != "" {
		return sshcfg.AgentKey{}, fmt.Errorf("%s: Secretive's agent lists %s, which the question did not offer: "+
			"run the wizard again to pick it, or to make a new one", macKey, key.Label())
	}
	return k.wait(ctx, now, what)
}

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
	text, _, err := loadConf(k.file())
	if err != nil {
		return err
	}
	keys, _, _ := setup.SecretiveKeys(ctx, k.M, k.Home)
	got, err := k.made(ctx, "a P-256 key for ws's login, at its default protection", keys)
	if err != nil {
		return err
	}
	if err := setup.Apply(ctx, k.M, k.Out, setup.KeyCopy{Path: sshcfg.Expand(got.Pub(), k.Home), Text: got.Line + "\n"}); err != nil {
		return err
	}
	if !hasLine(text, "WS_IDENTITY_PUB_KEY") {
		lines := line("WS_IDENTITY_PUB_KEY", got.Pub()) + "\n" + line("WS_IDENTITY_AGENT", k.tilde(k.secretive())+"/socket.ssh") + "\n"
		if err := refused(k.file().add(ctx, lines)); err != nil {
			return err
		}
	}
	_, c, err := loadConf(k.file())
	if err != nil {
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
