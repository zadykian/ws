package main

import (
	"cmp"
	"context"
	"sync"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// varsReach is the way to ws of the host_vars sheet. It starts with the sheet's first read, so that
// mac.conf and the secrets draw with ws out of reach.
type varsReach struct {
	p process
	// mu guards lock and release: the work takes the lock, and the command lets it go, past a work
	// that the screen's end left behind.
	mu      sync.Mutex
	lock    string
	release func()
}

// reach finds how the files script reaches root on ws, as the wizard's stages do. The lock keeps a
// wizard's own writes of the file away, and holds ssh's sockets.
func (r *varsReach) reach(ctx context.Context, b *tui.Bridge) (*wizard.Remote, error) {
	lock, err := r.locked(ctx)
	if err != nil {
		return nil, err
	}
	m := screenMachine(ctx, b, r.p.machine)
	conf := confPath(r.p.getenv, r.p.pwd)
	ssh := wizard.SSH{M: m, Lock: lock, Home: r.p.getenv("HOME"), Conf: conf}
	sudo := &wizard.Sudo{Host: &wizard.HostKey{Tty: r.p.tty, SSH: ssh, Out: b}, P: b}
	plan, err := sudo.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if len(plan.Changes) > 0 {
		if err := sudo.Apply(ctx, plan); err != nil {
			return nil, err
		}
	}
	return &wizard.Remote{SSH: ssh, Script: mac.RemoteFilesScript, Sudo: sudo}, nil
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
