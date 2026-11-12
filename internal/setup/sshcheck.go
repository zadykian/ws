package setup

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
)

// daemonLabel is the LaunchDaemon of the tunnel.
const daemonLabel = "com.github.zadykian.ws-tunnel"

// PickKey takes the key that mac.conf names for ws's login, as ReadKey reads it.
func PickKey(ctx context.Context, m effect.Machine, c conf.Conf, home string) (sshcfg.Identity, error) {
	k, err := ReadKey(ctx, m, c, home, LoginKey)
	return sshcfg.Identity{Pub: k.Pub, Agent: k.Agent, Text: k.Text}, err
}

// cldTabs is whether Ghostty's app is there and the cld in ~/.local/bin has restore tabs. With it,
// ssh never runs a cld that would refuse restore tabs --watch at each connection.
func (s SSH) cldTabs(ctx context.Context) bool {
	there := false
	for _, app := range []string{"/Applications/Ghostty.app", s.Home + "/Applications/Ghostty.app"} {
		if info, err := s.M.Stat(app); err == nil && info.Mode.IsDir() {
			there = true
		}
	}
	if !there {
		return false
	}
	cld := s.Home + "/.local/bin/cld"
	info, err := s.M.Stat(cld)
	if err != nil || info.Mode&0o111 == 0 {
		return false
	}
	_, err = s.M.Run(ctx, cld, "help", "restore", "tabs")
	return err == nil
}
