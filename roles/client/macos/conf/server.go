package conf

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
)

// FirstServer is the name of the one server that ws sets up for now.
const FirstServer = "ws"

// ServerPath returns the path of the server called name's file, in a directory beside mac.conf.
func ServerPath(mac, name string) string {
	return path.Join(path.Dir(mac), "servers", name, "server.conf")
}

// Load reads mac.conf at mac and a server's file at server, and parses each by its scope. It
// returns them joined: the Mac's keys from mac.conf, every other from the server's file. It
// returns every refusal of both files, each in its own file's words, then what the server's file
// takes that another server's file takes too ([clashes]).
func Load(m effect.Machine, mac, server string) (Conf, error) {
	macConf, macErr := load(m, mac, MacFile)
	c, serverErr := load(m, server, ServerFile)
	if err := errors.Join(macErr, serverErr); err != nil {
		return Conf{}, err
	}
	others, err := Siblings(m, mac, server)
	if err != nil {
		return Conf{}, err
	}
	var all []error
	for _, e := range clashes(c, others) {
		all = append(all, errors.New(e.In(server)))
	}
	if err := errors.Join(all...); err != nil {
		return Conf{}, err
	}
	return join(macConf, c), nil
}

// join returns the keys of a server's file with the Mac's keys from mac.conf.
func join(mac, server Conf) Conf {
	server.GitName, server.GitEmail, server.GitSigningPubKey = mac.GitName, mac.GitEmail, mac.GitSigningPubKey
	server.IdentityPubKey, server.IdentityAgent = mac.IdentityPubKey, mac.IdentityAgent
	return server
}

// load reads the file at file as its scope says.
func load(m effect.Machine, file string, s Scope) (Conf, error) {
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return Conf{}, fmt.Errorf("%s is missing: roles/client/macos/README.md says what goes in it", file)
	}
	if err != nil {
		return Conf{}, err
	}
	c, errs := s.Parse(string(text))
	var all []error
	for _, e := range errs {
		all = append(all, errors.New(e.In(file)))
	}
	return c, errors.Join(all...)
}
