package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/sshcfg"
)

// daemonLabel is the LaunchDaemon of the tunnel.
const daemonLabel = "com.github.zadykian.ws-tunnel"

// identity picks the key that ssh offers ws and checks its .pub. It warns where Secretive's key
// takes the place of the file key that the old config.d/ws gave ws.
func (s SSH) identity(ctx context.Context, f sshFacts) (sshcfg.Identity, error) {
	id, err := PickKey(ctx, s.M, s.Conf, s.Home)
	if err != nil || !id.Check {
		return id, err
	}
	if s.Conf.Identity == "" && strings.Contains(string(f.Ws.Text), "\n    IdentityFile ~/.ssh/id_ed25519\n") {
		s.Out.Warn("Secretive's key replaces ~/.ssh/id_ed25519 as ws's: keep the file key with "+
			"WS_IDENTITY=~/.ssh/id_ed25519.pub in mac.conf for an installed ws, or put the key of %s on GitHub "+
			"before ws's next install", id.Pub)
	}
	return id, nil
}

// PickKey picks the key that ssh offers ws from mac.conf and Secretive's keys, and checks its .pub
// where mac.conf or Secretive names one.
func PickKey(ctx context.Context, m effect.Machine, c conf.Conf, home string) (sshcfg.Identity, error) {
	secretive := home + "/" + sshcfg.SecretiveData
	var keys []string
	info, err := m.Stat(secretive + "/socket.ssh")
	socket := err == nil && info.Mode&fs.ModeSocket != 0
	if socket && c.Identity == "" {
		if keys, err = m.ReadDir(secretive + "/PublicKeys"); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return sshcfg.Identity{}, err
		}
		keys = files(m, secretive+"/PublicKeys", keys)
	}
	id, err := sshcfg.PickIdentity(c.Identity, c.IdentityAgent, c.GitSigningKey, home, socket, keys)
	if err != nil || !id.Check {
		return id, err
	}
	file := sshcfg.Expand(id.Pub, home)
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return id, fmt.Errorf("%s, mac.conf's WS_IDENTITY, is missing: make the key first", file)
	}
	if err != nil {
		return id, fmt.Errorf("cannot read %s: make it readable, or name another in mac.conf's WS_IDENTITY", file)
	}
	if err := sshcfg.CheckKeyFile(file, string(text)); err != nil {
		return id, err
	}
	if _, err := m.Run(ctx, "ssh-keygen", "-l", "-f", file); err != nil {
		return id, fmt.Errorf("ssh-keygen cannot read %s: name the .pub of a P-256 or ed25519 key in "+
			"mac.conf's WS_IDENTITY", file)
	}
	return id, nil
}

// viaTunnel is whether ws goes through the tunnel: where mac.conf has WS_PUBLIC_KEY and launchd has
// the daemon. A run that stops before the tunnel leaves ssh ws on the ways of ws-ssh.
func (s SSH) viaTunnel(ctx context.Context) bool {
	if s.Conf.PublicKey == "" {
		return false
	}
	if s.AssumeTunnel {
		return true
	}
	_, err := s.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel)
	return err == nil
}

// cldTabs is whether Ghostty's app is there and the cld in ~/.local/bin has restore tabs, so that
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

// files keeps the names in dir that are files, or links to files: the old script skipped a
// directory named like a key.
func files(m effect.Machine, dir string, names []string) []string {
	var found []string
	for _, name := range names {
		if info, err := m.Stat(dir + "/" + name); err == nil && info.Mode.IsRegular() {
			found = append(found, name)
		}
	}
	return found
}
