package keys

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// made has the user make what in Secretive's app, installed where missing, and returns the key
// once the agent lists it beside the keys of before.
func (k *MacKey) made(ctx context.Context, what string, before []sshcfg.AgentKey) (sshcfg.AgentKey, error) {
	if !k.app() {
		if err := k.M.Pass(ctx, false, "brew", "install", "--cask", "secretive"); err != nil {
			return sshcfg.AgentKey{}, fmt.Errorf("%s: brew install --cask secretive failed: install Secretive, "+
				"then run the wizard again", stage.Key)
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
			"run the wizard again to pick it, or to make a new one", stage.Key, key.Label())
	}
	return k.wait(ctx, now, what)
}

// open opens Secretive's app, or says to by hand.
func (k *MacKey) open(ctx context.Context) {
	if _, err := k.M.Run(ctx, "open", "-a", "Secretive"); err != nil {
		k.Out.Say("%s: warning: open cannot find Secretive: open it by hand", stage.Key)
	}
}
