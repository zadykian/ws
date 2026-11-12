package setup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
)

// tunnelPlist is where launchd reads the daemon's plist.
const tunnelPlist = "/Library/LaunchDaemons/" + daemonLabel + ".plist"

// Launchctl has launchd run or signal the daemon, through sudo.
type Launchctl struct {
	Args []string
	// Signal marks `kill SIGHUP`, which a daemon that is not running refuses: kickstart starts it
	// then, and it reads tunnel.conf as it starts.
	Signal bool
	// Routes marks the SIGHUP that has the daemon read routes.d again. A daemon that is not running
	// reads it as it starts, so a failure warns.
	Routes bool
}

func (l Launchctl) String() string { return "would run launchctl " + strings.Join(l.Args, " ") }

// launchctlAction picks launchd's action, a pure function of what differs. It bootstraps where the
// daemon isn't loaded, and boots it out and in where the plist changed. It kicks it where a program
// or the key changed. It sends SIGHUP where tunnel.conf alone changed, which the daemon takes
// without dropping the session.
func launchctlAction(loaded, plist, program, key, conf bool) []Launchctl {
	target := "system/" + daemonLabel
	switch {
	case !loaded:
		return []Launchctl{{Args: []string{"bootstrap", "system", tunnelPlist}}}
	case plist:
		return []Launchctl{{Args: []string{"bootout", target}}, {Args: []string{"bootstrap", "system", tunnelPlist}}}
	case program || key:
		return []Launchctl{{Args: []string{"kickstart", "-k", target}}}
	case conf:
		return []Launchctl{{Args: []string{"kill", "SIGHUP", target}, Signal: true}}
	}
	return nil
}

// daemon plans tunnel.conf, the plist and launchd's action, where mac.conf has WS_PUBLIC_KEY.
func (t *Tunnel) daemon(ctx context.Context) ([]step.Change, error) {
	cs, err := plan(ctx, nil, host.File{M: t.M, Path: tunnelDir + "/tunnel.conf", Content: []byte(tunnelConf(t.Conf)), Mode: 0o600})
	if err != nil {
		return nil, err
	}
	conf := len(cs)
	if cs, err = plan(ctx, cs, host.File{M: t.M, Path: tunnelPlist, Content: t.Plist, Mode: 0o644}); err != nil {
		return nil, err
	}
	plist := len(cs) > conf
	_, err = t.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel)
	for _, l := range launchctlAction(err == nil, plist, t.copied, t.keyMade, conf > 0) {
		cs = append(cs, l)
	}
	return cs, nil
}

// launchctl runs the action. A bootstrap right after a bootout may find the service still on its
// way out, so it tries for 10 s.
func launchctl(ctx context.Context, m effect.Machine, out effect.Out, sleep func(time.Duration), l Launchctl) error {
	var err error
	for tries := 1; ; tries++ {
		_, err = m.RunRoot(ctx, "launchctl", l.Args...)
		if err == nil {
			break
		}
		if l.Routes {
			out.Warn("cannot signal %s, which reads routes.d when it next starts", daemonLabel)
			return nil
		}
		if l.Signal {
			return launchctl(ctx, m, out, sleep, Launchctl{Args: []string{"kickstart", "-k", "system/" + daemonLabel}})
		}
		if l.Args[0] != "bootstrap" || tries >= 10 {
			return fmt.Errorf("launchctl %s failed: %w; is ws-tunnel on under System Settings › "+
				"General › Login Items & Extensions, Allow in the Background?", strings.Join(l.Args, " "), err)
		}
		sleep(time.Second)
	}
	out.Say("ran launchctl %s", strings.Join(l.Args, " "))
	return nil
}
