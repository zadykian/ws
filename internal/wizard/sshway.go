package wizard

import (
	"bytes"
	"context"
	"errors"
	"strings"
)

// Way is what ssh -G prints for ws at its direct way.
type Way struct {
	Host, Port, User, KnownHosts string
	// Inet tells that ssh takes IPv4 alone.
	Inet bool
}

// errNoWay says that no direct way leads to ws: mac.conf gives none and the Host key stage asked no
// address.
var errNoWay = errors.New("ssh finds no direct way to ws")

// Way reads ssh's settings for ws at its direct way, as argv gives it: the host and port that ssh
// reaches, and known_hosts. It fails where no way is given, and where ssh -G fails.
func (s SSH) Way(ctx context.Context) (Way, error) {
	if _, found := s.way(ctx); !found {
		return Way{}, errNoWay
	}
	out, err := s.M.Run(ctx, "ssh", append([]string{"-G"}, s.argv(ctx, []string{"ws-ssh"})...)...)
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
	return w, nil
}
