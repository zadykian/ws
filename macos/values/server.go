package values

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
)

// FirstServer is the name of the one server that ws sets up for now.
const FirstServer = "ws"

// ServerPath returns the path of the server called name's file, in a directory beside values.yml.
func ServerPath(mac, name string) string {
	return path.Join(path.Dir(mac), "servers", name, "values.yml")
}

// Load reads values.yml at mac and a server's file at server, and parses each by its scope. It
// returns them joined: the Mac's keys from values.yml, every other from the server's file. It
// returns every refusal of both files, each in its own file's words, then what the server's file
// takes that another server's file takes too ([clashes]).
func Load(m effect.Machine, mac, server string) (Conf, error) {
	macConf, macErr := load(m, mac, MacFile.Parse)
	c, serverErr := load(m, server, ServerFile.Parse)
	if err := errors.Join(macErr, serverErr); err != nil {
		return Conf{}, err
	}
	if err := refuseClashes(m, mac, server, c); err != nil {
		return Conf{}, err
	}
	return join(macConf, c), nil
}

// refuseClashes returns a refusal for what c, the file of the server at server, shares with another
// server's file beside values.yml at mac. That is a network, a domain, a slot, a port or the cable.
// The refusals use c's file's words.
func refuseClashes(m effect.Machine, mac, server string, c Conf) error {
	others, err := Siblings(m, mac, server)
	if err != nil {
		return err
	}
	var all []error
	for _, e := range clashes(c, others) {
		all = append(all, errors.New(e.In(server)))
	}
	return errors.Join(all...)
}

// LoadLines reads values.yml at mac and a server's file at server as Load does, but checks each line
// alone, as Scope.Lines does. A pair of keys may then stand apart. Each file refuses the other's
// keys, and no other server is read.
func LoadLines(m effect.Machine, mac, server string) (Conf, error) {
	macConf, macErr := load(m, mac, MacFile.Lines)
	c, serverErr := load(m, server, ServerFile.Lines)
	if err := errors.Join(macErr, serverErr); err != nil {
		return Conf{}, err
	}
	return join(macConf, c), nil
}

// join returns the keys of a server's file with the Mac's keys from values.yml.
func join(mac, server Conf) Conf {
	server.GitSigningPubKey, server.IdentityPubKey, server.IdentityAgent = mac.GitSigningPubKey, mac.IdentityPubKey, mac.IdentityAgent
	return server
}

// load reads the file at file, which parse checks.
func load(m effect.Machine, file string, parse func(string) (Conf, []Error)) (Conf, error) {
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return Conf{}, fmt.Errorf("%s is missing: macos/values/README.md says what goes in it", file)
	}
	if err != nil {
		return Conf{}, err
	}
	c, errs := parse(string(text))
	var all []error
	for _, e := range errs {
		all = append(all, errors.New(e.In(file)))
	}
	return c, errors.Join(all...)
}

// LoadMac reads values.yml at mac alone, which takes the Mac's keys alone: the keys of a Mac with
// no server yet, or of every server.
func LoadMac(m effect.Machine, mac string) (Conf, error) { return load(m, mac, MacFile.Parse) }
