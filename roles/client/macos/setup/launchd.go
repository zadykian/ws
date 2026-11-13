package setup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// daemonLabel is the LaunchDaemon of the tunnel.
const daemonLabel = "com.github.zadykian.ws-tunnel"

// tunnelDir holds the Mac's key, tunnel.conf and routes.d, which only root can write.
const tunnelDir = "/etc/ws-tunnel"

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

// Kind is restart.
func (Launchctl) Kind() step.ChangeKind { return step.Restart }

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
