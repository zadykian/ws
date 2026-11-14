// Package launchd runs launchctl through sudo for the LaunchDaemon of the tunnel.
package launchd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Dir holds the Mac's key and routes.d, and a directory of tunnel.conf per server, which only root can write.
const Dir = "/etc/ws-tunnel"

// Launchctl has launchd run or signal the daemon, through sudo.
type Launchctl struct {
	M   effect.Machine
	Out effect.Out
	// Sleep waits between the tries of a bootstrap. Nil means time.Sleep.
	Sleep func(time.Duration)
	Args  []string
	// Label is the daemon that Args act on, which a warning and the fallback to kickstart name.
	Label string
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

// Apply runs the action. A bootstrap right after a bootout may find the service still on its
// way out, so it tries for 10 s.
func (l Launchctl) Apply(ctx context.Context) error {
	sleep := l.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var err error
	for tries := 1; ; tries++ {
		_, err = l.M.RunRoot(ctx, "launchctl", l.Args...)
		if err == nil {
			break
		}
		if l.Routes {
			l.Out.Warn("cannot signal %s, which reads routes.d when it next starts", l.Label)
			return nil
		}
		if l.Signal {
			return Launchctl{M: l.M, Out: l.Out, Args: []string{"kickstart", "-k", "system/" + l.Label}}.Apply(ctx)
		}
		if l.Args[0] != "bootstrap" || tries >= 10 {
			return fmt.Errorf("launchctl %s failed: %w; is ws-tunnel on under System Settings › "+
				"General › Login Items & Extensions, Allow in the Background?", strings.Join(l.Args, " "), err)
		}
		sleep(time.Second)
	}
	l.Out.Say("ran launchctl %s", strings.Join(l.Args, " "))
	return nil
}
