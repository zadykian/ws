package tunnel

import (
	"bytes"
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// launchctlAction picks launchd's action, a pure function of what differs. It bootstraps where the
// daemon isn't loaded, and boots it out and in where the plist changed. It kicks it where a program
// or the key changed. It sends SIGHUP where tunnel.conf alone changed, which the daemon takes
// without dropping the session.
func launchctlAction(d launchd.Daemon, loaded, plist, program, key, conf bool) []launchd.Launchctl {
	target := "system/" + d.Label
	switch {
	case !loaded:
		return []launchd.Launchctl{{Args: []string{"bootstrap", "system", d.Plist}}}
	case plist:
		return []launchd.Launchctl{{Args: []string{"bootout", target}}, {Args: []string{"bootstrap", "system", d.Plist}}}
	case program || key:
		return []launchd.Launchctl{{Args: []string{"kickstart", "-k", target}}}
	case conf:
		return []launchd.Launchctl{{Args: []string{"kill", "SIGHUP", target}, Signal: true}}
	}
	return nil
}

// daemon plans tunnel.conf, the plist and launchd's action, where mac.conf has WS_PUBLIC_KEY. A
// changed tunnel.conf has the other loaded daemons read again: they count its endpoints too.
func (t *Tunnel) daemon(ctx context.Context) ([]step.Change, error) {
	d := launchd.For(t.Server)
	cs, err := resource.Append(ctx, nil, t.said(file.Dir{M: t.M, Path: d.Dir, Mode: 0o700}))
	if err == nil {
		cs, err = resource.Append(ctx, cs, t.said(file.Dir{M: t.M, Path: d.Routes, Mode: 0o755}))
	}
	if err == nil {
		cs, err = resource.Append(ctx, cs, t.said(file.File{M: t.M, Path: d.Conf, Content: []byte(tunnelConf(t.Conf)), Mode: 0o600}))
	}
	if err != nil {
		return nil, err
	}
	conf := len(cs)
	if cs, err = resource.Append(ctx, cs, t.said(file.File{M: t.M, Path: d.Plist, Content: t.plist(), Mode: 0o644})); err != nil {
		return nil, err
	}
	plist := len(cs) > conf
	_, err = t.M.Run(ctx, "launchctl", "print", "system/"+d.Label)
	for _, l := range launchctlAction(d, err == nil, plist, t.copied, t.keyMade, conf > 0) {
		l.M, l.Out, l.Sleep, l.Label = t.M, t.Out, t.sleep, d.Label
		cs = append(cs, l)
	}
	if conf == 0 {
		return cs, nil
	}
	others, err := launchd.Reread(ctx, t.M, t.Out, t.Server)
	return append(cs, others...), err
}

// plist returns the template of the plist with the server's name in it. A name is of a-z, 0-9 and
// -, which XML needs no escape for.
func (t *Tunnel) plist() []byte { return bytes.ReplaceAll(t.Plist, []byte("@NAME@"), []byte(t.Server)) }
