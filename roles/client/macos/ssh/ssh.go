// Package ssh sets up the Mac's ~/.ssh/config.d/ws and the Include that reads it.
package ssh

import (
	"context"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// SSH writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. It stops where
// config.d/ws is a file it did not write, and where the Include could not go first. It also stops
// where the key that ssh offers ws is wrong, and where ssh refuses the new file, alone or as ssh
// reads it whole.
type SSH struct {
	Conf conf.Conf
	// Server is the name of the server that the run acts for.
	Server string
	// MacConf is mac.conf's path, whose servers/ holds the other servers' files.
	MacConf string
	Home    string
	M       effect.Machine
	Out     effect.Out
	Now     func() time.Time
	// Again is the second run of the step, which cld may have changed ssh's file for.
	Again bool
}

// Step returns the step: ssh's parts, made as it plans from what ~/.ssh and the Mac show then.
func (s SSH) Step() step.Step {
	lazy := resource.Lazy{Title: "ssh", Parts: s.parts}
	if !s.Again {
		return lazy
	}
	lazy.Title, lazy.After = "ssh again", "ssh"
	return secondRun{lazy}
}

// secondRun is the step's second run, which manages the same files as the first and so reports
// nothing. It waits on ssh: where ssh is left out with changes, its plan would hold them too. The
// run's order puts it past cld.
type secondRun struct{ resource.Lazy }

// Report says nothing: the first run reports the same files.
func (secondRun) Report(context.Context) ([]step.Item, error) { return nil, nil }

// config returns the text of config.d/ws, a block for each of hosts.
func (s SSH) config(ctx context.Context, f sshFacts, key keys.NamedKey, hosts []sshcfg.Host) []byte {
	return []byte(sshcfg.Config(sshcfg.Inputs{
		Hosts:       hosts,
		Patterns:    sshcfg.IgnoreUnknown(string(f.UserConfig.Text), "UseKeychain"),
		IdentityPub: key.Pub, IdentityAgent: key.Agent, CldTabs: s.cldTabs(ctx),
	}))
}
