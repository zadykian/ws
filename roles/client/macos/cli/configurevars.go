package cli

import (
	"cmp"
	"context"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"sync"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/tui"
)

// varsReach is the way to ws of the host_vars and secrets sheets, of the wizard's state screen and of
// ws client macos secrets. It starts with the first read, so that mac.conf draws with ws out of
// reach.
type varsReach struct {
	p enginecli.Env
	// server is the name of the server that it reaches.
	server string
	// sudo is the scan's, whose way to root the state screen keeps. It stands unread where the scan
	// never reached it, and a read then asks. Nil makes a new one for each reach.
	sudo *host.Sudo
	// mu guards lock and release: the work takes the lock, and the command lets it go, past a work
	// that the screen's end left behind.
	mu      sync.Mutex
	lock    string
	release func()
}

// reach finds how the session reaches root on ws, as the wizard's stages do, through the screen.
func (r *varsReach) reach(ctx context.Context, b *tui.Bridge) (*session.Session, error) {
	return r.reachOn(ctx, b.Effects().M, b)
}

// rootTalk asks sudo's password and says what sudo answered: a screen's Bridge, or the terminal.
type rootTalk interface {
	effect.Hidden
	effect.Out
}

// reachOn is reach on m, which asks through t. The lock keeps a wizard's own run away, and holds
// ssh's sockets.
func (r *varsReach) reachOn(ctx context.Context, m effect.Machine, t rootTalk) (*session.Session, error) {
	lock, err := r.locked(ctx)
	if err != nil {
		return nil, err
	}
	macConf := conf.Path(r.p.Getenv, r.p.Pwd)
	server := conf.ServerPath(macConf, r.server)
	ssh := remote.SSH{M: m, Lock: lock, Home: r.p.Getenv("HOME"), Host: r.server,
		Ways: host.MacWays{M: m, Conf: server}}
	sudo := cmp.Or(r.sudo, &host.Sudo{})
	// A kept one holds the effects of a screen that has ended.
	ws := session.New(host.MacReach{SSH: ssh, Sudo: sudo, Conf: server, Out: t},
		session.Scripts{Remote: session.RemoteScript, Files: session.RemoteFilesScript})
	sudo.SSH, sudo.Tty, sudo.P, sudo.Out = ssh, r.p.TTY, t, t
	plan, err := sudo.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if len(plan.Changes) > 0 {
		if err := sudo.Apply(ctx, plan); err != nil {
			return nil, err
		}
	}
	return ws, nil
}

// locked takes the wizard's lock, once.
func (r *varsReach) locked(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.release != nil {
		return r.lock, nil
	}
	lock, release, err := wizard.Lock(ctx, r.p.Machine, r.p.LockDir())
	if err != nil {
		return "", err
	}
	r.lock, r.release = lock, release
	return lock, nil
}

// close ends ssh's masters and lets the lock go, where a read took it. A run that a signal cut must
// do so all the same.
func (r *varsReach) close(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.release != nil {
		remote.SSH{M: r.p.Machine, Lock: r.lock}.End(context.WithoutCancel(ctx))
		r.release()
	}
}
