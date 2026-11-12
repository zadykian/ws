package main

import (
	"cmp"
	"context"
	"sync"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// varsReach is the way to ws of the host_vars and secrets sheets, of the wizard's state screen and of
// ws mac secrets. It starts with the first read, so that mac.conf draws with ws out of reach.
type varsReach struct {
	p process
	// sudo is the scan's, whose way to root the state screen keeps. It stands unread where the scan
	// never reached it, and a read then asks. Nil makes a new one for each reach.
	sudo *wizard.Sudo
	// mu guards lock and release: the work takes the lock, and the command lets it go, past a work
	// that the screen's end left behind.
	mu      sync.Mutex
	lock    string
	release func()
}

// sessionOpen finds the session to ws of a sheet or a read, through the screen: the Mac's over ssh,
// or ws's own.
type sessionOpen func(ctx context.Context, b *tui.Bridge) (*wizard.Session, error)

// hostVarsSheet is the host_vars form before its read, which open reaches ws for. The password that
// sudo asks stays with the session, so that a save asks none.
func hostVarsSheet(open sessionOpen) tui.Sheet {
	return tui.Sheet{
		Name:   "host_vars",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			ws, err := open(ctx, b)
			if err != nil {
				return tui.Sheet{}, err
			}
			form, err := wizard.VarsForm{Ws: ws}.Load(ctx)
			if err != nil {
				return tui.Sheet{}, err
			}
			return varsSheet(form), nil
		},
	}
}

// reach finds how the session reaches root on ws, as the wizard's stages do, through the screen.
func (r *varsReach) reach(ctx context.Context, b *tui.Bridge) (*wizard.Session, error) {
	return r.reachOn(ctx, b.Effects().M, b)
}

// rootTalk asks sudo's password and says what sudo answered: a screen's Bridge, or the terminal.
type rootTalk interface {
	effect.Hidden
	effect.Out
}

// reachOn is reach on m, which asks through t. The lock keeps a wizard's own run away, and holds
// ssh's sockets.
func (r *varsReach) reachOn(ctx context.Context, m effect.Machine, t rootTalk) (*wizard.Session, error) {
	lock, err := r.locked(ctx)
	if err != nil {
		return nil, err
	}
	conf := confPath(r.p.getenv, r.p.pwd)
	ssh := wizard.SSH{M: m, Lock: lock, Home: r.p.getenv("HOME"), Conf: conf}
	sudo := cmp.Or(r.sudo, &wizard.Sudo{})
	// A kept one holds the effects of a screen that has ended.
	ws := wizard.NewSession(wizard.MacReach{SSH: ssh, Sudo: sudo, Out: t},
		wizard.Scripts{Remote: mac.RemoteScript, Files: mac.RemoteFilesScript})
	sudo.SSH, sudo.Tty, sudo.P, sudo.Out = ssh, r.p.tty, t, t
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
	lock, release, err := wizard.Lock(ctx, r.p.machine, r.p.lockDir())
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
		wizard.SSH{M: r.p.machine, Lock: r.lock}.End(context.WithoutCancel(ctx))
		r.release()
	}
}

// varsSheet is ws's host_vars on the Configure screen, as form read it.
func varsSheet(form wizard.VarsForm) tui.Sheet {
	return tui.Sheet{
		Name:   "host_vars",
		Fields: func(map[string]string) []tui.FieldSpec { return fieldSpecs(form.Fields()) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			next, err := form.Save(ctx, b, edits)
			return varsSheet(next), err
		},
	}
}
