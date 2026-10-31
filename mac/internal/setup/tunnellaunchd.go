package setup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// tunnelPlist is where launchd reads the daemon's plist.
const tunnelPlist = "/Library/LaunchDaemons/" + daemonLabel + ".plist"

// Launchctl has launchd run or signal the daemon, through sudo.
type Launchctl struct {
	Args []string
	// Signal marks `kill SIGHUP`, which a daemon that is not running refuses: kickstart starts it
	// then, and it reads tunnel.conf as it starts.
	Signal bool
}

func (l Launchctl) String() string { return "would run launchctl " + strings.Join(l.Args, " ") }

// launchctlAction picks launchd's action, a pure function of what differs. It bootstraps where the
// daemon isn't loaded, boots it out and in where the plist changed, kicks it where a program or the
// key changed, and sends SIGHUP where tunnel.conf alone changed, which the daemon takes without
// dropping the session.
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
	var cs []step.Change
	conf, err := rootFile(t.M, tunnelEtc+"/tunnel.conf", 0o600, []byte(tunnelConf(t.Conf)))
	if err != nil {
		return nil, err
	}
	plistText, err := t.M.ReadFile(t.Plist)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", t.Plist, err)
	}
	plist, err := rootFile(t.M, tunnelPlist, 0o644, plistText)
	if err != nil {
		return nil, err
	}
	for _, c := range []step.Change{conf, plist} {
		if c != nil {
			cs = append(cs, c)
		}
	}
	_, err = t.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel)
	for _, l := range launchctlAction(err == nil, plist != nil, t.copied, t.keyMade, conf != nil) {
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
