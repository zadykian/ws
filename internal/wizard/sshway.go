package wizard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
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

// login tells whether ssh logs in to ws at w as WS_USER by the Mac's key, and runs a command. It
// connects only where known_hosts holds a key under ws. ssh's error comes back in one line, as ssh
// ends each line with \r\n and an agent's refusal to sign comes before its last line.
func (s SSH) login(ctx context.Context, w Way) (ok bool, sshErr string) {
	if !KnownHost(ctx, s.M, w.KnownHosts) {
		return false, ""
	}
	out, err := s.Run(ctx, "-o", "BatchMode=yes", "ws", "echo", "ok")
	if err == nil && strings.TrimSpace(string(out)) == "ok" {
		return true, ""
	}
	if err == nil {
		return false, ""
	}
	return false, Stderr(err)
}

// KnownHost tells whether known_hosts, the file, holds a key under ws, on m.
func KnownHost(ctx context.Context, m effect.Machine, file string) bool {
	_, err := m.Run(ctx, "ssh-keygen", "-F", "ws", "-f", file)
	return err == nil
}

// Way reads ssh's settings for ws at its direct way, as argv gives it: the host and port that ssh
// reaches, and known_hosts. It fails where no way is given, and where ssh -G fails.
func (s SSH) Way(ctx context.Context) (Way, error) {
	if _, _, found := s.way(ctx); !found {
		return Way{}, s.noWay()
	}
	out, err := s.M.Run(ctx, "ssh", append([]string{"-G"}, s.argv(ctx, []string{"ws"})...)...)
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

// noWay says why no way leads to ws: a mac.conf that is missing or refused gives none, and it names
// that rather than ssh.
func (s SSH) noWay() error {
	_, err := macConf(s.M, s.Conf)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("ws is out of reach: no mac.conf at %s; the wizard or ws client macos configure writes it", s.Conf)
	case err != nil:
		return fmt.Errorf("ws is out of reach: mac.conf leaves no way: %w", err)
	}
	return errNoWay
}
