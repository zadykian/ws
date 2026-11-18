package ssh

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/internal/etcws"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// The modes of ssh's files.
const (
	sshDirMode  fs.FileMode = 0o700
	sshFileMode fs.FileMode = 0o600
)

// includeLine is the line that puts /etc/ws/ssh_config first in ~/.ssh/config.
const includeLine = "Include " + sshcfg.File

// sshFacts is what the ssh step read of the Mac.
type sshFacts struct {
	M    effect.Machine
	Out  effect.Out
	Home string
	// Dirs plans ~/.ssh.
	Dirs       []step.Change
	UserConfig file.Found // ~/.ssh/config, whose text is the file a link points to
	// IncludeLine is the line of its Include of /etc/ws/ssh_config, or 0.
	IncludeLine int
	// Config plans ~/.ssh/config, or Stop says why a run can't. Include is how it stands.
	Config  []step.Change
	Stop    error
	Include step.Item
}

// sshDirs is ~/.ssh alone: the hosts are in /etc/ws, and a run leaves ~/.ssh/config.d be.
func sshDirs(m effect.Machine, out effect.Out, home string) file.UserDirs {
	return file.UserDirs{{M: m, Out: out, Path: home + "/.ssh", Mode: sshDirMode}}
}

// refusal is a part that plans no change: it stops a plan, or warns, as a report never does.
type refusal func(ctx context.Context) error

// Plan runs the refusal.
func (r refusal) Plan(ctx context.Context) ([]resource.Change, error) { return nil, r(ctx) }

// parts reads ~/.ssh and the Mac, and returns the parts of the step in the order of its changes.
// They are the refusal, the key's copy, ~/.ssh, /etc/ws and its ssh_config, the Include and the
// copy's report.
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
	for _, part := range keys.CopyParts(s.M, s.Out, key.File, key.Text) {
		copies = append(copies, part)
	}
	copied, err := keys.PlanCopy(ctx, s.M, s.Out, key.File, key.Text)
	parts := []resource.Resource{
		s.refusal(f, hosts, text),
		resource.Planned{Changes: copied, Err: err},
		resource.Planned{Changes: f.Dirs, Seen: sshDirs(s.M, s.Out, s.Home)},
	}
	files := etcws.Files
	if len(copies) > 0 {
		files = etcws.Inside // the copy's parts make /etc/ws, and a plan holds it once
	}
	for _, r := range files(s.M, sshcfg.File, text) {
		parts = append(parts, resource.Said{Out: s.Out, Part: r})
	}
	return append(parts,
		resource.Planned{Changes: f.Config, Seen: file.Fixed{f.Include}},
		resource.Planned{Seen: copies},
	), nil
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

// ownHost stops where a Host of the user's own config names a server. The line is counted in
// ~/.ssh/config as it stands.
func ownHost(text string, hosts []sshcfg.Host) error {
	if own := sshcfg.OwnHosts(text, names(hosts)...); len(own) > 0 {
		return fmt.Errorf("line %d of ~/.ssh/config, %s, sets its own values for %s: remove it, as "+
			"/etc/ws/ssh_config holds each server's host", own[0].Line, own[0].Text, own[0].Names)
	}
	return nil
}

// refusal stops where ssh can't take the files, or where the user's own Host names a server, and
// warns of what else the run leaves to the user.
func (s SSH) refusal(f sshFacts, hosts []sshcfg.Host, text []byte) refusal {
	return func(ctx context.Context) error {
		if f.Stop != nil {
			return f.Stop
		}
		if err := ownHost(string(f.UserConfig.Text), hosts); err != nil {
			return err
		}
		u := userConfig{Exists: f.UserConfig.Exists, Text: string(f.UserConfig.Text), IncludeLine: f.IncludeLine}
		return validate(ctx, s.M, s.Home, u, hosts, text)
	}
}
