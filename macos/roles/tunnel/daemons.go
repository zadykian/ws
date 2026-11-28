package tunnel

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/launchd"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/etcws"
)

// Dir holds the Mac's key and a directory per server, root's and readable by all but the key.
const Dir = etcws.Dir + "/tunnel"

// daemons is where launchd reads plists, and LabelPrefix starts a tunnel daemon's label, the
// prefix of launchd's unit of the tunnel.
const (
	daemons     = "/Library/LaunchDaemons"
	LabelPrefix = "com.github.zadykian.ws-tunnel."
)

// YieldFile holds the networks that each daemon yields to another owner, a network a line. It
// sits in /var/run, as a reboot that clears it takes the other owner's routes too.
const YieldFile = "/var/run/ws-tunnel.yield"

// Daemon is one server's tunnel daemon: its launchd job and the files it keeps.
type Daemon struct{ Name, Label, Plist, Dir, Routes, Conf, State, Log string }

// For returns the daemon of the server called name. Routes is the directory in Dir of the networks
// that this server's tunnel takes.
func For(name string) Daemon {
	label := LabelPrefix + name
	dir := Dir + "/" + name
	return Daemon{
		Name: name, Label: label, Plist: daemons + "/" + label + ".plist",
		Dir: dir, Routes: dir + "/routes.d", Conf: dir + "/tunnel.conf",
		State: "/var/run/ws-tunnel." + name + ".state", Log: "/var/log/ws-tunnel." + name + ".log",
	}
}

// Installed returns the names of the servers whose daemon has a plist, in order. The directory is
// readable by all, so it needs no root.
func Installed(m effect.Machine) ([]string, error) {
	entries, err := m.ReadDir(daemons)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if name, ok := nameIn(e, LabelPrefix, ".plist"); ok {
			names = append(names, name)
		}
	}
	return names, nil
}

// nameIn returns the server's name between prefix and suffix in a file's name. A name holds no dot,
// so a file of an older form, or one that ends in .new, gives none.
func nameIn(entry, prefix, suffix string) (string, bool) {
	name, ok := strings.CutPrefix(entry, prefix)
	if !ok {
		return "", false
	}
	if name, ok = strings.CutSuffix(name, suffix); ok && name != "" && !strings.Contains(name, ".") {
		return name, true
	}
	return "", false
}

// Found returns, in order, the names of the servers with a trace of a daemon on the Mac. A trace is
// a plist, a state file, a log, or a directory under Dir. Found inverts For. Each directory is
// readable by all, so it needs no root. A name holds no dot, so private.key falls out.
func Found(m effect.Machine) ([]string, error) {
	names, err := Installed(m)
	if err != nil {
		return nil, err
	}
	for _, from := range []struct{ dir, prefix, suffix string }{
		{"/var/run", "ws-tunnel.", ".state"},
		{"/var/log", "ws-tunnel.", ".log"},
		{Dir, "", ""},
	} {
		entries, err := m.ReadDir(from.dir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, e := range entries {
			if name, ok := nameIn(e, from.prefix, from.suffix); ok {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

// Reread plans a SIGHUP of each daemon that launchd has loaded, except those of the servers in
// skip, so that each reads routes.d and the other servers' files again. A daemon skips a network
// that another server's routes.d holds, so a change to one reaches them all.
func Reread(ctx context.Context, m effect.Machine, out effect.Out, skip ...string) ([]step.Change, error) {
	names, err := Installed(m)
	if err != nil {
		return nil, err
	}
	var cs []step.Change
	for _, name := range names {
		if label := For(name).Label; !slices.Contains(skip, name) && loaded(ctx, m, label) {
			cs = append(cs, reread{launchd.Launchctl{M: m}, out, label})
		}
	}
	return cs, nil
}

// loaded is whether launchd has the daemon loaded, which print shows the user with no root.
func loaded(ctx context.Context, m effect.Machine, label string) bool {
	_, err := m.Run(ctx, "launchctl", "print", "system/"+label)
	return err == nil
}

// reread has a loaded daemon read routes.d and the other servers' files again, by SIGHUP. A daemon
// that runs no process reads them as it starts, so a refusal warns.
type reread struct {
	l     launchd.Launchctl
	out   effect.Out
	label string
}

func (r reread) String() string { return "send SIGHUP to system/" + r.label }

// Kind is restart.
func (reread) Kind() step.ChangeKind { return step.Restart }

// Apply signals the daemon through sudo.
func (r reread) Apply(ctx context.Context) error {
	if err := r.l.Signal(ctx, r.label, "SIGHUP"); err != nil {
		r.out.Warn("cannot signal %s, which reads routes.d when it next starts", r.label)
	}
	return nil
}
