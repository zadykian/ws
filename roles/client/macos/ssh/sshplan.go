package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// The modes of ssh's files.
const (
	sshDirMode  fs.FileMode = 0o700
	sshFileMode fs.FileMode = 0o600
)

// includeLine is the line that puts config.d/ws first in ~/.ssh/config.
const includeLine = "Include config.d/ws"

// sshFacts is what the ssh step read of the Mac.
type sshFacts struct {
	M    effect.Machine
	Out  effect.Out
	Home string
	// Dirs plans ~/.ssh and ~/.ssh/config.d, and DirLink is the last of them that is a link.
	Dirs       []step.Change
	DirLink    file.UserDirLink
	Ws         file.Found // ~/.ssh/config.d/ws
	UserConfig file.Found // ~/.ssh/config, whose text is the file a link points to
	// IncludeLine is the line of its Include of config.d/ws, or 0.
	IncludeLine int
	// Config plans ~/.ssh/config, or Stop says why a run can't. Include is how it stands.
	Config  []step.Change
	Stop    error
	Include step.Item
}

// sshDirs is ~/.ssh and its config.d. A run sets the mode of config.d, which holds the hosts.
func sshDirs(m effect.Machine, out effect.Out, home string) file.UserDirs {
	return file.UserDirs{
		{M: m, Out: out, Path: home + "/.ssh", Mode: sshDirMode},
		{M: m, Out: out, Path: home + "/.ssh/config.d", Mode: sshDirMode, Fix: true},
	}
}

// refusal is a part that plans no change: it stops a plan, or warns, as a report never does.
type refusal func(ctx context.Context) error

// Plan runs the refusal.
func (r refusal) Plan(ctx context.Context) ([]resource.Change, error) { return nil, r(ctx) }

// parts reads ~/.ssh and the Mac, and returns the parts of the step in the order of its changes.
// They are the refusal, the key's copy, the directories, config.d/ws, the Include and the copy's
// report.
func (s SSH) parts(ctx context.Context) ([]resource.Resource, error) {
	f, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	key, err := keys.ReadKey(ctx, s.M, s.Conf, s.Home, keys.LoginKey)
	if err != nil {
		return nil, err
	}
	hosts, err := s.hosts()
	if err != nil {
		return nil, err
	}
	text := s.config(ctx, f, key, hosts)
	var copies reports
	if keys.Copied(key.File, key.Text, s.Home) {
		copies = append(copies, keys.CopyDirs(s.M, s.Out, s.Home))
	}
	copies = append(copies, keys.CopyReport(s.M, s.Out, key.File, key.Text, s.Home)...)
	copied, err := keys.PlanCopy(s.M, s.Out, key.File, key.Text, s.Home)
	return []resource.Resource{
		s.refusal(f, hosts, text),
		resource.Planned{Changes: copied, Err: err},
		resource.Planned{Changes: f.Dirs, Seen: sshDirs(s.M, s.Out, s.Home)},
		resource.Planned{Changes: s.wsFile(text).Change(f.Ws), Seen: s.wsFile(text)},
		resource.Planned{Changes: f.Config, Seen: file.Fixed{f.Include}},
		resource.Planned{Seen: copies},
	}, nil
}

// reports is the files that the step reports, in order.
type reports []step.Reporter

// Report gathers the items of the files.
func (r reports) Report(ctx context.Context) ([]step.Item, error) { return file.Gather(ctx, r...) }

// names is the name of each host.
func names(hosts []sshcfg.Host) []string {
	out := make([]string, len(hosts))
	for i, h := range hosts {
		out[i] = h.Name
	}
	return out
}

// refusal stops where ssh can't take the files, or where the user's own Host names a server, and
// warns of what else the run leaves to the user.
func (s SSH) refusal(f sshFacts, hosts []sshcfg.Host, text []byte) refusal {
	return func(ctx context.Context) error {
		if f.Stop != nil {
			return f.Stop
		}
		// The line is counted in the text the run leaves, so that each Plan names the same line.
		own := sshcfg.OwnHosts(string(s.configLines().After(f.UserConfig, f.IncludeLine > 0)), names(hosts)...)
		if len(own) > 0 {
			return fmt.Errorf("line %d of ~/.ssh/config, %s, sets its own values for %s: remove it, as "+
				"config.d/ws holds each server's host", own[0].Line, own[0].Text, own[0].Names)
		}
		if err := s.validate(ctx, f, hosts, text); err != nil {
			return err
		}
		if f.DirLink.Path == s.Home+"/.ssh/config.d" && !bytes.Equal(f.Ws.Text, text) {
			s.Out.Warn("~/.ssh/config.d is a link to %s: config.d/ws goes there", f.DirLink.To)
		}
		return nil
	}
}
