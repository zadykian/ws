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

func (k *MacKey) secretive() string { return k.Home + "/" + sshcfg.SecretiveData }

// Plan finds what the key that ssh offers ws lacks. A file key, or a key where mac.conf has no line
// for WS_IDENTITY_PUB_KEY, is checked here; a Secretive key's agent must list it.
func (k *MacKey) Plan(ctx context.Context) (step.Plan, error) {
	text, c, err := loadConf(k.file())
	if err != nil {
		return step.Plan{}, err
	}
	cs, err := k.changes(ctx, text, c)
	if err != nil {
		return step.Plan{}, err
	}
	git, err := k.gitQuestion(ctx, c)
	return step.Plan{Changes: append(cs, git...)}, err
}

func (k *MacKey) changes(ctx context.Context, text string, c conf.Conf) ([]step.Change, error) {
	// A WS_IDENTITY_PUB_KEY that is no key of Secretive's, with no agent, is a key in a file.
	if c.IdentityPubKey != "" && c.IdentityAgent == "" {
		if pub := sshcfg.Expand(c.IdentityPubKey, k.Home); !sshcfg.SecretiveKey(pub, k.Home) {
			return nil, k.fileKey(ctx, strings.TrimSuffix(pub, ".pub"))
		}
	}
	id, err := setup.PickKey(ctx, k.M, c, k.Home)
	if err != nil {
		return nil, err
	}
	pub, agent := sshcfg.Expand(id.Pub, k.Home), sshcfg.Expand(id.Agent, k.Home)
	var cs []step.Change
	switch key := strings.TrimSuffix(pub, ".pub"); {
	case agent != "":
		if !k.listed(ctx, id.Text, agent) {
			return nil, fmt.Errorf("%s: the agent at %s lists no key of %s: open Secretive, where it is that agent, "+
				"or start the agent, then run the wizard again", macKey, agent, pub)
		}
		// The stages after this one offer the key, and ssh reads its .pub for that.
		cs = append(cs, setup.PlanCopy(k.M, pub, id.Text, k.Home)...)
	case !k.exists(key) && !k.exists(pub):
		choose := chooseKey{pub: pub, app: k.app()}
		if !choose.app {
			return []step.Change{choose}, nil
		}
		cs = append(cs, choose)
	default:
		if err := k.fileKey(ctx, key); err != nil {
			return nil, err
		}
		// A Secretive key made later would take the file key's place.
		if !hasLine(text, "WS_IDENTITY_PUB_KEY") {
			after := text
			if after != "" && !strings.HasSuffix(after, "\n") {
				after += "\n"
			}
			return []step.Change{setup.Write{Path: k.Conf, Before: []byte(text), After: []byte(after + line("WS_IDENTITY_PUB_KEY", id.Pub) + "\n"),
				Exists: text != "", Mode: 0o644}}, nil
		}
	}
	return cs, nil
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
