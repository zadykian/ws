package launchd

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// daemons is where launchd reads plists, and labelPrefix starts a tunnel daemon's label.
const (
	daemons     = "/Library/LaunchDaemons"
	labelPrefix = "com.github.zadykian.ws-tunnel."
)

// Daemon is one server's tunnel daemon: its launchd job and the files it keeps.
type Daemon struct{ Name, Label, Plist, Dir, Conf, State, Log string }

// For returns the daemon of the server called name. Dir is root's.
func For(name string) Daemon {
	label := labelPrefix + name
	dir := Dir + "/" + name
	return Daemon{
		Name: name, Label: label, Plist: daemons + "/" + label + ".plist",
		Dir: dir, Conf: dir + "/tunnel.conf",
		State: "/var/run/ws-tunnel." + name + ".state", Log: "/var/log/ws-tunnel." + name + ".log",
	}
}

// Installed returns the names of the servers whose daemon has a plist, in order. The directory is
// world-readable, so it needs no root.
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
		name, ok := strings.CutPrefix(e, labelPrefix)
		if !ok {
			continue
		}
		if name, ok = strings.CutSuffix(name, ".plist"); ok && name != "" && !strings.Contains(name, ".") {
			names = append(names, name)
		}
	}
	return names, nil
}
