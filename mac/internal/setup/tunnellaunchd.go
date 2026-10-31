package setup

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Launchctl has launchd run or signal the daemon, through sudo.
type Launchctl struct {
	Args []string
	// Signal marks `kill SIGHUP`, which a daemon that is not running refuses: kickstart starts it
	// then, and it reads tunnel.conf as it starts.
	Signal bool
}

func (l Launchctl) String() string { return "would run launchctl " + strings.Join(l.Args, " ") }

// launchctl runs the action. A bootstrap right after a bootout may find the service still on its
// way out, so it tries for 10 s.
func (t *Tunnel) launchctl(ctx context.Context, l Launchctl) error {
	var err error
	for tries := 1; ; tries++ {
		_, err = t.M.RunRoot(ctx, "launchctl", l.Args...)
		if err == nil {
			break
		}
		if l.Signal {
			return t.launchctl(ctx, Launchctl{Args: []string{"kickstart", "-k", "system/" + daemonLabel}})
		}
		if l.Args[0] != "bootstrap" || tries >= 10 {
			return fmt.Errorf("launchctl %s failed: %w; is ws-tunnel on under System Settings › "+
				"General › Login Items & Extensions, Allow in the Background?", strings.Join(l.Args, " "), err)
		}
		t.sleep(time.Second)
	}
	t.Out.Say("ran launchctl %s", strings.Join(l.Args, " "))
	return nil
}
