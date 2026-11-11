package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/step"
)

func (k *MacKey) file() confFile { return confFile{k.Conf, macKey, k.M, k.Out} }

// Plan finds what the key that ssh offers ws lacks: a question where mac.conf names none, or ws's
// copy of a key of Secretive's. A file key must be there, and an agent's key listed by its agent.
func (k *MacKey) Plan(ctx context.Context) (step.Plan, error) {
	_, c, err := loadConf(k.file())
	if err != nil {
		return step.Plan{}, err
	}
	cs, err := k.changes(ctx, c)
	if err != nil {
		return step.Plan{}, err
	}
	git, err := k.gitQuestion(ctx, c)
	return step.Plan{Changes: append(cs, git...)}, err
}

// changes checks ws's key where mac.conf names it, and asks for one where it names none, an empty
// value included.
func (k *MacKey) changes(ctx context.Context, c conf.Conf) ([]step.Change, error) {
	if c.IdentityPubKey != "" {
		return k.named(ctx, c)
	}
	keys, _, err := setup.SecretiveKeys(ctx, k.M, k.Home)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", macKey, err)
	}
	return []step.Change{askKey{loginUse, keyQuestion(loginUse, keys, k.login(c))}}, nil
}

// named checks the key that mac.conf names for ws. ws's copy of a key of Secretive's comes from
// the agent, which must list the key.
func (k *MacKey) named(ctx context.Context, c conf.Conf) ([]step.Change, error) {
	if pub := sshcfg.Expand(c.IdentityPubKey, k.Home); c.IdentityAgent == "" && !sshcfg.SecretiveKey(pub, k.Home) {
		return nil, k.fileKey(ctx, strings.TrimSuffix(pub, ".pub"))
	}
	id, err := setup.PickKey(ctx, k.M, c, k.Home)
	if err != nil {
		return nil, err
	}
	pub, agent := sshcfg.Expand(id.Pub, k.Home), sshcfg.Expand(id.Agent, k.Home)
	if !k.listed(ctx, id.Text, agent) {
		return nil, fmt.Errorf("%s: the agent at %s lists no key of %s: open Secretive, where it is that agent, "+
			"or start the agent, then run the wizard again", macKey, agent, pub)
	}
	// The stages after this one offer the key, and ssh reads its .pub for that.
	return setup.PlanCopy(k.M, pub, id.Text, k.Home), nil
}

// gitQuestion asks for git's key after ws's key's changes, whichever, where git has a name, no key.
func (k *MacKey) gitQuestion(ctx context.Context, c conf.Conf) ([]step.Change, error) {
	if c.GitName == "" || c.GitSigningPubKey != "" {
		return nil, nil
	}
	keys, _, err := setup.SecretiveKeys(ctx, k.M, k.Home)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", macKey, err)
	}
	return []step.Change{askKey{gitUse, keyQuestion(gitUse, keys, k.login(c))}}, nil
}
