package setup

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/step"
)

// SSH writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. It stops where
// config.d/ws is a file it did not write, and where the Include could not go first. It also stops
// where the key that ssh offers ws is wrong, and where ssh refuses the new file, alone or as ssh
// reads it whole.
type SSH struct {
	Conf conf.Conf
	Home string
	M    effect.Machine
	Out  effect.Out
	Now  func() time.Time
	// Again is the second run of the step, which cld may have changed ssh's file for.
	Again bool
}

// Name is ssh, or ssh again.
func (s SSH) Name() string {
	if s.Again {
		return "ssh again"
	}
	return "ssh"
}

// WaitsOn is ssh for the second run: where ssh is left out with changes, its plan would hold them
// too. The run's order puts it past cld.
func (s SSH) WaitsOn() string {
	if s.Again {
		return "ssh"
	}
	return ""
}

// Plan reads ~/.ssh and the Mac, and checks the new file with ssh.
func (s SSH) Plan(ctx context.Context) (step.Plan, error) {
	f, err := s.read()
	if err != nil {
		return step.Plan{}, err
	}
	id, err := s.identity(ctx, f)
	if err != nil {
		return step.Plan{}, err
	}
	text := s.config(ctx, f, id)
	if err := s.validate(ctx, f, text); err != nil {
		return step.Plan{}, err
	}
	if f.ConfigDirLink != "" && !bytes.Equal(f.Ws.Text, text) {
		s.Out.Warn("~/.ssh/config.d is a link to %s: config.d/ws goes there",
			f.ConfigDirLink)
	}
	for _, h := range sshcfg.OwnHosts(string(f.UserConfig.Text), "ws") {
		s.Out.Warn("line %d of ~/.ssh/config, %s, sets its own values for %s: those in config.d/ws, "+
			"read first, win where both set a key", h.Line, h.Text, h.Names)
	}
	copied := PlanCopy(s.M, sshcfg.Expand(id.Pub, s.Home), id.Text, s.Home)
	return step.Plan{Changes: append(copied, sshChanges(f, text)...)}, nil
}

// config returns the text of config.d/ws.
func (s SSH) config(ctx context.Context, f sshFacts, id sshcfg.Identity) []byte {
	return []byte(sshcfg.Config(sshcfg.Inputs{
		User: s.Conf.User, Patterns: sshcfg.IgnoreUnknown(string(f.UserConfig.Text), "UseKeychain"),
		IdentityPub: id.Pub, IdentityAgent: id.Agent, CldTabs: s.cldTabs(ctx),
	}))
}

// Report says how ~/.ssh stands against the plan. It picks the key as Plan does, past Plan's
// warning. The step's second run manages the same files, so it says nothing.
func (s SSH) Report(ctx context.Context) ([]step.Item, error) {
	if s.Again {
		return nil, nil
	}
	f, err := s.read()
	if err != nil {
		return nil, err
	}
	id, err := PickKey(ctx, s.M, s.Conf, s.Home)
	if err != nil {
		return nil, err
	}
	dir := s.Home + "/.ssh"
	there := f.UserConfig.Exists
	include := lineItem(dir+"/config", includeLine, there, there && !f.UserConfigNeedsInclude)
	if own := sshcfg.OwnHosts(string(f.UserConfig.Text), "ws"); len(own) > 0 {
		include.Conflict = fmt.Sprintf("line %d sets its own values for ws: those in config.d/ws, read "+
			"first, win where both set a key", own[0].Line)
	}
	parts := []step.Reporter{
		userPath{M: s.M, Kind: step.DirItem, Path: dir},
		userPath{M: s.M, Kind: step.DirItem, Path: dir + "/config.d", Mode: sshDirMode},
		userFile{M: s.M, Path: dir + "/config.d/ws", Text: s.config(ctx, f, id), Mode: sshFileMode},
		fixed{include},
	}
	return gather(ctx, append(parts, keyCopy(s.M, sshcfg.Expand(id.Pub, s.Home), id.Text, s.Home)...)...)
}

// Apply applies the changes in order.
func (s SSH) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, s.M, s.Out, c); err != nil {
			return err
		}
	}
	return nil
}
