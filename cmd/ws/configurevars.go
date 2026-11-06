package main

import (
	"cmp"
	"context"
	"sync"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// varsReach is the way to ws of the host_vars sheet and of the wizard's state screen. It starts with
// the first read, so that mac.conf and the secrets draw with ws out of reach.
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

// sheet is the host_vars form before its read.
func (r *varsReach) sheet() tui.Sheet {
	return tui.Sheet{
		Name:   "host_vars",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open:   r.open,
	}
}

// open reaches ws and reads the file. The password that sudo asks stays with the form, so that a
// save asks none.
func (r *varsReach) open(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
	files, err := r.reach(ctx, b, mac.RemoteFilesScript)
	if err != nil {
		return tui.Sheet{}, err
	}
	form, err := wizard.VarsForm{Files: files}.Load(ctx)
	if err != nil {
		return tui.Sheet{}, err
	}
	return varsSheet(form), nil
}

// reach finds how script reaches root on ws, as the wizard's stages do. The lock keeps a wizard's
// own run away, and holds ssh's sockets.
func (r *varsReach) reach(ctx context.Context, b *tui.Bridge, script []byte) (*wizard.Remote, error) {
	lock, err := r.locked(ctx)
	if err != nil {
		return nil, err
	}
	m := screenMachine(ctx, b, r.p.machine)
	conf := confPath(r.p.getenv, r.p.pwd)
	ssh := wizard.SSH{M: m, Lock: lock, Home: r.p.getenv("HOME"), Conf: conf}
	sudo := cmp.Or(r.sudo, &wizard.Sudo{})
	// A kept one holds the effects of a screen that has ended.
	sudo.Host, sudo.P = &wizard.HostKey{Tty: r.p.tty, SSH: ssh, Out: b}, b
	plan, err := sudo.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if len(plan.Changes) > 0 {
		if err := sudo.Apply(ctx, plan); err != nil {
			return nil, err
		}
	}
	return &wizard.Remote{SSH: ssh, Script: script, Sudo: sudo}, nil
}

// locked takes the wizard's lock, once.
func (r *varsReach) locked(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.release != nil {
		return r.lock, nil
	}
	lock, release, err := wizard.Lock(ctx, r.p.machine, cmp.Or(r.p.getenv("TMPDIR"), "/tmp"))
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
