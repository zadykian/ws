package setup

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/roles/client/macos/conf"
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
	f, err := s.read(ctx)
	if err == nil {
		err = f.Stop
	}
	if err != nil {
		return step.Plan{}, err
	}
	key, err := ReadKey(ctx, s.M, s.Conf, s.Home, LoginKey)
	if err != nil {
		return step.Plan{}, err
	}
	text := s.config(ctx, f, key)
	if err := s.validate(ctx, f, text); err != nil {
		return step.Plan{}, err
	}
	if f.DirLink.Path == s.Home+"/.ssh/config.d" && !bytes.Equal(f.Ws.Text, text) {
		s.Out.Warn("~/.ssh/config.d is a link to %s: config.d/ws goes there", f.DirLink.To)
	}
	// The line is counted in the text the run leaves, so that the runner's second Plan names the
	// same line and the warning shows once.
	for _, h := range sshcfg.OwnHosts(string(s.configLines().after(f.UserConfig, f.IncludeLine > 0)), "ws") {
		s.Out.Warn("line %d of ~/.ssh/config, %s, sets its own values for %s: those in config.d/ws, "+
			"read first, win where both set a key", h.Line, h.Text, h.Names)
	}
	copied, err := PlanCopy(s.M, key.File, key.Text, s.Home)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: append(copied, sshChanges(f, text)...)}, nil
}

// config returns the text of config.d/ws.
func (s SSH) config(ctx context.Context, f sshFacts, key NamedKey) []byte {
	return []byte(sshcfg.Config(sshcfg.Inputs{
		User: s.Conf.User, Patterns: sshcfg.IgnoreUnknown(string(f.UserConfig.Text), "UseKeychain"),
		IdentityPub: key.Pub, IdentityAgent: key.Agent, CldTabs: s.cldTabs(ctx),
	}))
}

// Report says how ~/.ssh stands against the plan, with the key that Plan takes. The step's second
// run manages the same files, so it says nothing.
func (s SSH) Report(ctx context.Context) ([]step.Item, error) {
	if s.Again {
		return nil, nil
	}
	f, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	key, err := ReadKey(ctx, s.M, s.Conf, s.Home, LoginKey)
	if err != nil {
		return nil, err
	}
	include := f.Include
	if own := sshcfg.OwnHosts(string(f.UserConfig.Text), "ws"); len(own) > 0 && include.Conflict == "" {
		include.Conflict = fmt.Sprintf("line %d sets its own values for ws: those in config.d/ws, read "+
			"first, win where both set a key", own[0].Line)
	}
	parts := []step.Reporter{
		sshDirs(s.M, s.Home),
		s.wsFile(s.config(ctx, f, key)),
		fixed{include},
	}
	if copied(key.File, key.Text, s.Home) {
		parts = append(parts, copyDirs(s.M, s.Home))
	}
	return gather(ctx, append(parts, keyCopy(s.M, key.File, key.Text, s.Home)...)...)
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
