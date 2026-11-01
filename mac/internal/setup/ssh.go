package setup

import (
	"bytes"
	"context"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/sshcfg"
	"github.com/zadykian/ws/mac/internal/step"
)

// SSH writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. It stops where
// config.d/ws is a file it did not write, where the Include could not go first, where the key that
// ssh offers ws is wrong, and where ssh refuses the new file, alone or as ssh reads it whole.
type SSH struct {
	Conf conf.Conf
	Home string
	M    effect.Machine
	Out  effect.Out
	Now  func() time.Time
	// AssumeTunnel has ws go through the tunnel wherever mac.conf has WS_PUBLIC_KEY, as under --check,
	// which does not load the daemon.
	AssumeTunnel bool
	// Again is the second run of the step, which the tunnel and cld may have changed ssh's file for.
	Again bool
}

// Name is ssh, or ssh again.
func (s SSH) Name() string {
	if s.Again {
		return "ssh again"
	}
	return "ssh"
}

// WaitsOn is the tunnel for the second run.
func (s SSH) WaitsOn() string {
	if s.Again {
		return "tunnel"
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
	in := sshcfg.Inputs{
		User: s.Conf.User, LAN: s.Conf.LAN, LANAddress: s.Conf.LANAddress, Remote: s.Conf.Remote,
		RemotePort: s.Conf.RemotePort, Patterns: sshcfg.IgnoreUnknown(string(f.UserConfig.Text), "UseKeychain"),
		IdentityPub: id.Pub, IdentityAgent: id.Agent, ViaTunnel: s.viaTunnel(ctx), CldTabs: s.cldTabs(ctx),
	}
	text := []byte(sshcfg.Config(in))
	if err := s.validate(ctx, f, text); err != nil {
		return step.Plan{}, err
	}
	if f.ConfigDirLink != "" && !bytes.Equal(f.Ws.Text, text) {
		s.Out.Warn("~/.ssh/config.d is a link to %s: config.d/ws, with mac.conf's addresses, goes there",
			f.ConfigDirLink)
	}
	for _, h := range sshcfg.OwnHosts(string(f.UserConfig.Text), "ws", "ws-ssh") {
		s.Out.Warn("line %d of ~/.ssh/config, %s, sets its own values for %s: those in config.d/ws, "+
			"read first, win where both set a key", h.Line, h.Text, h.Names)
	}
	return step.Plan{Changes: sshChanges(f, text)}, nil
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
