package values

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
)

// loadServer reads the server's file at file, and gives a guest its host's way. The Mac's keys are
// not joined: Load does that.
func loadServer(m effect.Machine, mac, file string) (Conf, error) {
	c, err := load(m, file, ServerFile.Parse)
	if err != nil {
		return c, err
	}
	if msg := resolveHost(m, mac, file, &c); msg != "" {
		return Conf{}, errors.New(Error{Msg: msg}.In(file))
	}
	return c, nil
}

// resolveHost gives the guest c, whose file is at file, the way of its host: the LAN, the address
// on it, the router's name and the cable. A host that is no server, that is refused or that is a
// guest too gives the refusal. A server with no WS_HOST stays as it stands.
func resolveHost(m effect.Machine, mac, file string, c *Conf) string {
	if c.Host == "" {
		return ""
	}
	hostFile := ServerPath(mac, c.Host)
	if hostFile == file {
		return fmt.Sprintf("WS_HOST=%s names the guest's own server", c.Host)
	}
	if _, err := m.Stat(hostFile); errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf("WS_HOST=%s names no server: %s is missing", c.Host, hostFile)
	}
	h, err := load(m, hostFile, ServerFile.Parse)
	switch {
	case err != nil:
		return fmt.Sprintf("WS_HOST=%s names a server whose file is refused: %v", c.Host, err)
	case h.Host != "":
		return fmt.Sprintf("WS_HOST=%s names a guest: a host is a server with a way of its own", c.Host)
	}
	c.Target = "guest"
	c.LAN, c.LANAddress, c.Remote = h.LAN, h.LANAddress, h.Remote
	c.HostCable = h.CableService != ""
	return ""
}

// GuestsOf returns the servers beside values.yml at mac whose WS_HOST names host, in order. A file
// that is refused names none: the check says why.
func GuestsOf(m effect.Machine, mac, host string) ([]string, error) {
	names, err := Servers(m, mac)
	if err != nil {
		return nil, err
	}
	var guests []string
	for _, n := range names {
		if c, err := load(m, ServerPath(mac, n), ServerFile.Parse); err == nil && c.Host == host {
			guests = append(guests, n)
		}
	}
	return guests, nil
}
