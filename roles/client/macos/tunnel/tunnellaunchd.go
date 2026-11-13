package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// tunnelPlist is where launchd reads the daemon's plist.
const tunnelPlist = "/Library/LaunchDaemons/" + launchd.Label + ".plist"

// launchctlAction picks launchd's action, a pure function of what differs. It bootstraps where the
// daemon isn't loaded, and boots it out and in where the plist changed. It kicks it where a program
// or the key changed. It sends SIGHUP where tunnel.conf alone changed, which the daemon takes
// without dropping the session.
func launchctlAction(loaded, plist, program, key, conf bool) []launchd.Launchctl {
	target := "system/" + launchd.Label
	switch {
	case !loaded:
		return []launchd.Launchctl{{Args: []string{"bootstrap", "system", tunnelPlist}}}
	case plist:
		return []launchd.Launchctl{{Args: []string{"bootout", target}}, {Args: []string{"bootstrap", "system", tunnelPlist}}}
	case program || key:
		return []launchd.Launchctl{{Args: []string{"kickstart", "-k", target}}}
	case conf:
		return []launchd.Launchctl{{Args: []string{"kill", "SIGHUP", target}, Signal: true}}
	}
	return nil
}

// daemon plans tunnel.conf, the plist and launchd's action, where mac.conf has WS_PUBLIC_KEY.
func (t *Tunnel) daemon(ctx context.Context) ([]step.Change, error) {
	cs, err := resource.Append(ctx, nil, t.said(file.File{M: t.M, Path: launchd.Dir + "/tunnel.conf", Content: []byte(tunnelConf(t.Conf)), Mode: 0o600}))
	if err != nil {
		return nil, err
	}
	conf := len(cs)
	if cs, err = resource.Append(ctx, cs, t.said(file.File{M: t.M, Path: tunnelPlist, Content: t.Plist, Mode: 0o644})); err != nil {
		return nil, err
	}
	plist := len(cs) > conf
	_, err = t.M.Run(ctx, "launchctl", "print", "system/"+launchd.Label)
	for _, l := range launchctlAction(err == nil, plist, t.copied, t.keyMade, conf > 0) {
		l.M, l.Out, l.Sleep = t.M, t.Out, t.sleep
		cs = append(cs, l)
	}
	return cs, nil
}
