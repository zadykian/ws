package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
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

// PickKey picks the key that ssh offers ws from mac.conf and the keys of Secretive's agent, and
// checks its .pub where mac.conf or Secretive names one. A key that the agent lists needs no file:
// it comes as ws's copy, with the copy's text, which a later step writes.
func PickKey(ctx context.Context, m effect.Machine, c conf.Conf, home string) (sshcfg.Identity, error) {
	keys, socket, err := SecretiveKeys(ctx, m, home)
	if err != nil && c.Identity == "" {
		return sshcfg.Identity{}, err
	}
	names := make([]string, 0, len(keys))
	for _, k := range keys {
		names = append(names, k.Name)
	}
	id, err := sshcfg.PickIdentity(c.Identity, c.IdentityAgent, c.GitSigningKey, home, socket, names)
	if err != nil || !id.Check {
		return id, err
	}
	file := sshcfg.Expand(id.Pub, home)
	if k, ok := held(keys, file, home); ok {
		id.Pub, id.Text = k.Pub(), k.Line+"\n"
		return id, sshcfg.CheckKeyFile(sshcfg.Expand(id.Pub, home), k.Line)
	}
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return id, fmt.Errorf("%s, mac.conf's WS_IDENTITY, is missing: make the key first", file)
	}
	if errors.Is(err, fs.ErrPermission) && sshcfg.SecretiveKey(file, home) {
		return id, fmt.Errorf("cannot read %s, and Secretive's agent lists no key of it: open Secretive, "+
			"or name another in mac.conf's WS_IDENTITY", file)
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
	id.Text = string(text)
	return id, nil
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
