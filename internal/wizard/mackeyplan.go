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
// for WS_IDENTITY, is checked here; a Secretive key's agent must list it.
func (k *MacKey) Plan(ctx context.Context) (step.Plan, error) {
	text, c, err := loadConf(k.file())
	if err != nil {
		return step.Plan{}, err
	}
	cs, err := k.changes(ctx, text, c)
	return step.Plan{Changes: cs}, err
}

func (k *MacKey) changes(ctx context.Context, text string, c conf.Conf) ([]step.Change, error) {
	// A WS_IDENTITY outside Secretive's PublicKeys, with no agent, is a key in a file.
	if c.Identity != "" && c.IdentityAgent == "" {
		if pub := sshcfg.Expand(c.Identity, k.Home); !strings.HasPrefix(pub, k.secretive()+"/PublicKeys/") {
			return k.fileKey(ctx, strings.TrimSuffix(pub, ".pub"))
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
		if !k.listed(ctx, pub, agent) {
			return nil, fmt.Errorf("%s: the agent at %s lists no key of %s: open Secretive, where it is that agent, "+
				"or start the agent, then run the wizard again", macKey, agent, pub)
		}
	case !k.exists(key) && !k.exists(pub):
		choose := chooseKey{pub: pub, app: k.app()}
		if !choose.app {
			return []step.Change{choose}, nil
		}
		cs, agent = append(cs, choose), k.secretive()+"/socket.ssh"
	default:
		if _, err := k.fileKey(ctx, key); err != nil {
			return nil, err
		}
		// A Secretive key made later would take the file key's place.
		if !hasLine(text, "WS_IDENTITY") {
			after := text
			if after != "" && !strings.HasSuffix(after, "\n") {
				after += "\n"
			}
			return []step.Change{setup.Write{Path: k.Conf, Before: []byte(text), After: []byte(after + line("WS_IDENTITY", id.Pub) + "\n"),
				Exists: text != "", Mode: 0o644}}, nil
		}
	}
	// git, which has a name, needs a key of its own beside ws's in Secretive, unless mac.conf has a
	// line for WS_GIT_SIGNING_KEY. An empty one keeps ws mac apply's key.
	if agent == k.secretive()+"/socket.ssh" && c.GitName != "" && !hasLine(text, "WS_GIT_SIGNING_KEY") {
		cs = append(cs, gitKey{})
	}
	return cs, nil
}
