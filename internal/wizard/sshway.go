package wizard

import (
	"bytes"
	"context"
	"errors"
	"strings"
)

// Way is what ssh -G prints for ws-ssh.
type Way struct {
	Host, Port, User, KnownHosts string
	// Inet tells that ssh takes IPv4 alone.
	Inet bool
}

// errNoWay says that no way matched: ws-ssh is then its own host.
var errNoWay = errors.New("ssh finds no way to ws-ssh")

// Way reads ssh's settings for ws-ssh, which setup's config.d/ws gives, with HostName the address
// that the Host key stage asked: the host and port that ssh reaches, and known_hosts. It fails
// where no way matched.
func (s SSH) Way(ctx context.Context) (Way, error) {
	args := []string{"-G", "ws-ssh"}
	if first := s.First(); first != "" {
		args = []string{"-G", "-o", "HostName=" + first, "ws-ssh"}
	}
	out, err := s.M.Run(ctx, "ssh", args...)
	if err != nil {
		return Way{}, errNoWay
	}
	get := func(key string) string {
		for _, l := range bytes.Split(out, []byte("\n")) {
			f := strings.Fields(string(l))
			if len(f) > 1 && f[0] == key {
				return f[1]
			}
		}
		return ""
	}
	w := Way{Host: get("hostname"), Port: get("port"), User: get("user"),
		KnownHosts: get("userknownhostsfile"), Inet: get("addressfamily") == "inet"}
	if rest, ok := strings.CutPrefix(w.KnownHosts, "~/"); ok {
		w.KnownHosts = s.Home + "/" + rest
	}
	if w.Host == "ws-ssh" {
		return w, errNoWay
	}
	return w, nil
}
