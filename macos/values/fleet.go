package values

import (
	"errors"
	"fmt"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
)

// Server is one server of servers/ as the read of the Mac's values found it.
type Server struct {
	Name, File string
	// Conf holds the file's values, given a guest's host's way. Empty where the file did not parse.
	Conf Conf
	// Err is the server's refusal, in its own file's words, or nil: a bad file, a host that is
	// refused, or a value that another server holds too.
	Err error
	// fileErr is Err before the clashes between servers.
	fileErr error
	// ownErr is fileErr for the server's own file alone: no other server's file, no clash.
	ownErr error
}

// FileErr returns the refusal of the server's own file, which is missing, bad or names the server
// as its own host. It holds no other server's refusal, as its host's, and no clash.
func (s Server) FileErr() error { return s.ownErr }

// Fleet is each server's file in servers/, read once.
type Fleet struct {
	// NamesErr is the refusal of servers/ itself, such as a name that is bad.
	NamesErr error
	// Servers holds each server, accepted or refused, in order.
	Servers []Server
}

// parsedFile is one server's file as parsed, before its host's way is given.
type parsedFile struct {
	conf Conf
	err  error
}

// ReadFleet reads each server's file below dir, once, and works out each
// server's host and each clash between servers. It returns no error: each refusal sits in the
// field of the file that it names.
func ReadFleet(m effect.Machine, dir string) Fleet {
	var f Fleet
	var names []string
	names, f.NamesErr = Servers(m, dir)
	parsed := map[string]parsedFile{}
	for _, n := range names {
		c, err := load(m, ServerPath(dir, n), Parse)
		parsed[n] = parsedFile{c, err}
	}
	for _, n := range names {
		p := parsed[n]
		s := Server{Name: n, File: ServerPath(dir, n), fileErr: p.err, ownErr: p.err}
		if p.err == nil {
			s.Conf = p.conf
			if msg := resolveHost(&s.Conf, n, dir, parsed); msg != "" {
				s.fileErr = errors.New(Error{Msg: msg}.In(s.File))
				if s.Conf.Host == n {
					s.ownErr = s.fileErr
				}
			}
		}
		f.Servers = append(f.Servers, s)
	}
	return f.clash()
}

// resolveHost gives the guest c, the server called name, the way of its host: the LAN, the address
// on it, the router's name and the cable. A host that is no server, that is refused or that is a
// guest too gives the refusal. A server with no WS_HOST stays as it stands.
func resolveHost(c *Conf, name, dir string, parsed map[string]parsedFile) string {
	if c.Host == "" {
		return ""
	}
	h, ok := parsed[c.Host]
	switch {
	case c.Host == name:
		return fmt.Sprintf("WS_HOST=%s names the guest's own server", c.Host)
	case !ok:
		return fmt.Sprintf("WS_HOST=%s names no server: %s is missing", c.Host, ServerPath(dir, c.Host))
	case h.err != nil:
		return fmt.Sprintf("WS_HOST=%s names a server whose file is refused: %v", c.Host, h.err)
	case h.conf.Host != "":
		return fmt.Sprintf("WS_HOST=%s names a guest: a host is a server with a way of its own", c.Host)
	}
	c.Target = "guest"
	c.LAN, c.LANAddress, c.Remote = h.conf.LAN, h.conf.LANAddress, h.conf.Remote
	c.HostCable = h.conf.CableService != ""
	return ""
}

// clash sets each server's refusal: its file's, else what it takes that another server whose file
// is accepted takes too ([clashes]). Both servers refuse, so no order of servers decides which does.
func (f Fleet) clash() Fleet {
	var accepted []Sibling
	for _, s := range f.Servers {
		if s.fileErr == nil {
			accepted = append(accepted, Sibling{Name: s.Name, File: s.File, Conf: s.Conf})
		}
	}
	servers := slices.Clone(f.Servers)
	for i, s := range servers {
		s.Err = s.fileErr
		if s.Err == nil {
			var all []error
			for _, e := range clashes(s.Conf, slices.DeleteFunc(slices.Clone(accepted),
				func(o Sibling) bool { return o.Name == s.Name })) {
				all = append(all, errors.New(e.In(s.File)))
			}
			s.Err = errors.Join(all...)
		}
		servers[i] = s
	}
	f.Servers = servers
	return f
}

// Without returns the fleet as it stands once the server called name is gone, with the clashes of
// the others with it gone too.
func (f Fleet) Without(name string) Fleet {
	f.Servers = slices.DeleteFunc(slices.Clone(f.Servers), func(s Server) bool { return s.Name == name })
	return f.clash()
}

// Server returns the server called name, and whether the Mac has it.
func (f Fleet) Server(name string) (Server, bool) {
	i := slices.IndexFunc(f.Servers, func(s Server) bool { return s.Name == name })
	if i < 0 {
		return Server{}, false
	}
	return f.Servers[i], true
}

// Names returns the servers' names, in order.
func (f Fleet) Names() []string {
	names := make([]string, 0, len(f.Servers))
	for _, s := range f.Servers {
		names = append(names, s.Name)
	}
	return names
}

// Accepted returns the servers that nothing refuses.
func (f Fleet) Accepted() []Server {
	return slices.DeleteFunc(slices.Clone(f.Servers), func(s Server) bool { return s.Err != nil })
}

// Refused returns the refusal of each refused server, each worded as [RefuseSibling] does, or nil
// where none is refused.
func (f Fleet) Refused() error {
	var all []error
	for _, s := range f.Servers {
		if s.Err != nil {
			all = append(all, RefuseSibling(s.Name, s.Err, f.GuestsOf(s.Name)))
		}
	}
	return errors.Join(all...)
}

// Refusal returns every refusal of the read once: servers/'s and each server's, in the words of its
// own file.
func (f Fleet) Refusal() error {
	return errors.Join(f.NamesErr, f.Refused())
}

// GuestsOf returns the servers whose WS_HOST names host, in order. A file that does not parse
// names none: the check says why. The server called host is none of its own guests: its file is
// refused, and remove takes it.
func (f Fleet) GuestsOf(host string) []string {
	var guests []string
	for _, s := range f.Servers {
		if s.Conf.Host == host && s.Name != host {
			guests = append(guests, s.Name)
		}
	}
	return guests
}

// Load returns the values of the server called name, as a command for one server reads them. Where
// its file is refused, it returns the refusal in the file's words. Else it returns the server's
// clash, if any. No server's bad file refuses it but its own and, for a guest, its host's.
func (f Fleet) Load(name string) (Conf, error) {
	s, ok := f.Server(name)
	if !ok {
		return Conf{}, fmt.Errorf("no server named %s", name)
	}
	if s.fileErr != nil {
		return Conf{}, s.fileErr
	}
	return s.Conf, s.Err
}
