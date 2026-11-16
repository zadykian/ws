package keys

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func (k *MacKey) file() values.File {
	return values.File{Path: k.Conf, Stage: stage.Key, M: k.M, Out: k.Out}
}

// Plan finds what the key that ssh offers ws lacks: a question where values.yml names none, or ws's
// copy of a key of Secretive's. A file key must be there, and an agent's key listed by its agent.
func (k *MacKey) Plan(ctx context.Context) (step.Plan, error) {
	_, c, err := k.file().Load()
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

// changes checks ws's key where values.yml names it, and asks for one where it names none, an empty
// value included.
func (k *MacKey) changes(ctx context.Context, c values.Conf) ([]step.Change, error) {
	if c.IdentityPubKey != "" {
		return k.named(ctx, c)
	}
	keys, _, err := keys.SecretiveKeys(ctx, k.M, k.Home)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stage.Key, err)
	}
	return []step.Change{askKey{loginUse, keyQuestion(loginUse, keys, k.login(c))}}, nil
}

// named checks the key that values.yml names for ws. ws's copy of a key of Secretive's comes from
// the agent, which must list the key.
func (k *MacKey) named(ctx context.Context, c values.Conf) ([]step.Change, error) {
	if pub := sshcfg.Expand(c.IdentityPubKey, k.Home); c.IdentityAgent == "" && !sshcfg.SecretiveKey(pub, k.Home) {
		return nil, k.fileKey(ctx, strings.TrimSuffix(pub, ".pub"))
	}
	key, err := keys.ReadKey(ctx, k.M, c, k.Home, keys.LoginKey)
	if err != nil {
		return nil, err
	}
	pub, agent := key.File, sshcfg.Expand(key.Agent, k.Home)
	if !k.listed(ctx, key.Key, agent) {
		return nil, fmt.Errorf("%s: the agent at %s lists no key of %s: open Secretive, where it is that agent, "+
			"or start the agent, then run the wizard again", stage.Key, agent, pub)
	}
	// The stages after this one offer the key, and ssh reads its .pub for that.
	return keys.PlanCopy(ctx, k.M, k.Out, pub, key.Text)
}

// Asked reports whether the stage asks for a key. It asks for ws's where values.yml names none. It
// asks for git's where values.yml names none for it. An empty value names none.
func Asked(c values.Conf) bool { return c.IdentityPubKey == "" || gitAsked(c) }

func gitAsked(c values.Conf) bool { return c.GitSigningPubKey == "" }

// gitQuestion asks for git's key after ws's key's changes, whichever, where values.yml names none.
func (k *MacKey) gitQuestion(ctx context.Context, c values.Conf) ([]step.Change, error) {
	if !gitAsked(c) {
		return nil, nil
	}
	keys, _, err := keys.SecretiveKeys(ctx, k.M, k.Home)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stage.Key, err)
	}
	return []step.Change{askKey{gitUse, keyQuestion(gitUse, keys, k.login(c))}}, nil
}
