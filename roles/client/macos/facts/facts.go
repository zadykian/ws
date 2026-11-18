// Package facts reads what the Mac's specs are filled with: the user, the home and each server's
// values, which the Mac's ws unit hands to its units.
package facts

import (
	"errors"
	"fmt"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// Mac is the data of the Mac's specs. A spec names a field as {{ .User }}.
type Mac struct {
	// User is the account that ws runs as, and Home its home.
	User, Home string
	// Servers holds each server of servers/ beside values.yml, by name.
	Servers []Server
}

// Server is one server's values, joined with the Mac's own keys.
type Server struct {
	Name string
	values.Conf
}

// Read reads the facts from the process's environment and the values' files: values.yml from
// WS_VALUES or /etc/ws, and each server's file beside it. It runs the whole check of the values.
// So a bad value, a bad name or a clash between two servers fails every unit's read, and mise
// stops before a change. A Mac with no server yet reads none.
func Read(m effect.Machine, getenv func(string) string, pwd string) (Mac, error) {
	f := Mac{User: getenv("USER"), Home: getenv("HOME")}
	if f.User == "" || !path.IsAbs(f.Home) {
		return Mac{}, fmt.Errorf("USER %q and HOME %q name no user: run it from the user's own shell", f.User, f.Home)
	}
	mac := values.Path(getenv, pwd)
	names, err := values.Servers(m, mac)
	if err != nil {
		return Mac{}, err
	}
	var all []error
	for _, name := range names {
		c, err := values.Load(m, mac, values.ServerPath(mac, name))
		if err != nil {
			all = append(all, values.RefuseSibling(name, err))
			continue
		}
		f.Servers = append(f.Servers, Server{Name: name, Conf: c})
	}
	if err := errors.Join(all...); err != nil {
		return Mac{}, err
	}
	return f, nil
}
